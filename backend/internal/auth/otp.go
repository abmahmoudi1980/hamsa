package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/platform/httpx"
	"hamsa/internal/platform/sms"
)

// Clock abstracts time for testability.
type Clock interface{ Now() time.Time }

// RealClock is the production clock.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// OTP lifetimes and limits (research.md R4).
const (
	otpValidity       = 2 * time.Minute
	otpMaxAttempts    = 3
	otpResendThrottle = 60 * time.Second
)

// OTPCode mirrors the `otp_codes` table (migration 0001).
type OTPCode struct {
	ID         uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	Phone      string     `gorm:"column:phone"`
	CodeHash   string     `gorm:"column:code_hash"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	Attempts   int        `gorm:"column:attempts"`
	ConsumedAt *time.Time `gorm:"column:consumed_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
}

func (OTPCode) TableName() string { return "otp_codes" }

// OTPStore abstracts otp_codes persistence so the service is unit-testable
// without a database.
type OTPStore interface {
	// Latest returns the most recent code for phone (by created_at DESC), or
	// gorm.ErrRecordNotFound when none exists.
	Latest(ctx context.Context, phone string) (*OTPCode, error)
	Create(ctx context.Context, code *OTPCode) error
	Save(ctx context.Context, code *OTPCode) error
}

// GormOTPStore is the PostgreSQL implementation of OTPStore.
type GormOTPStore struct{ DB *gorm.DB }

func (s *GormOTPStore) Latest(ctx context.Context, phone string) (*OTPCode, error) {
	var code OTPCode
	if err := s.DB.WithContext(ctx).Where("phone = ?", phone).
		Order("created_at DESC").First(&code).Error; err != nil {
		return nil, err
	}
	return &code, nil
}

func (s *GormOTPStore) Create(ctx context.Context, code *OTPCode) error {
	return s.DB.WithContext(ctx).Create(code).Error
}

func (s *GormOTPStore) Save(ctx context.Context, code *OTPCode) error {
	return s.DB.WithContext(ctx).Save(code).Error
}

var iranianMobile = regexp.MustCompile(`^09\d{9}$`)

// OTP domain errors (Persian, per contracts/api.md).
var (
	ErrInvalidPhone    = httpx.BadRequest("شماره موبایل معتبر نیست.")
	ErrThrottled       = httpx.RateLimited("لطفاً کمی صبر کنید و دوباره درخواست کد کنید.")
	ErrOTPNotFound     = httpx.Unauthorized("کد تأیید یافت نشد. ابتدا کد را درخواست کنید.")
	ErrOTPExpired      = httpx.Unauthorized("کد تأیید منقضی شده است.")
	ErrTooManyAttempts = httpx.RateLimited("تعداد تلاش‌های مجاز به پایان رسیده است.")
	ErrInvalidCode     = httpx.Unauthorized("کد تأیید نادرست است.")
)

// OTPService issues and verifies one-time codes.
type OTPService struct {
	store  OTPStore
	sender sms.SmsSender
	clock  Clock
	dev    bool
}

// NewOTPService returns an OTP service. A nil clock defaults to RealClock and
// a nil sender defaults to the console sender.
func NewOTPService(store OTPStore, sender sms.SmsSender, clock Clock, dev bool) *OTPService {
	if clock == nil {
		clock = RealClock{}
	}
	if sender == nil {
		sender = &sms.ConsoleSender{}
	}
	return &OTPService{store: store, sender: sender, clock: clock, dev: dev}
}

// Issue creates, stores, and sends a fresh 6-digit code for phone. The code is
// returned only in dev mode; otherwise the returned string is empty.
func (s *OTPService) Issue(ctx context.Context, phone string) (string, error) {
	if !iranianMobile.MatchString(phone) {
		return "", ErrInvalidPhone
	}

	now := s.clock.Now()
	if latest, err := s.store.Latest(ctx, phone); err == nil &&
		now.Sub(latest.CreatedAt) < otpResendThrottle {
		return "", ErrThrottled
	}

	code, err := randomDigits(6)
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}

	otp := &OTPCode{
		ID:        uuid.New(),
		Phone:     phone,
		CodeHash:  hashCode(code),
		ExpiresAt: now.Add(otpValidity),
		Attempts:  0,
		CreatedAt: now,
	}
	if err := s.store.Create(ctx, otp); err != nil {
		return "", fmt.Errorf("store code: %w", err)
	}

	if err := s.sender.Send(phone, fmt.Sprintf("کد ورود شما به همسا: %s", code)); err != nil {
		return "", fmt.Errorf("send code: %w", err)
	}

	if s.dev {
		return code, nil
	}
	return "", nil
}

// Verify checks the submitted code against the latest code for phone and
// consumes it on success, enforcing expiry and the attempt limit.
func (s *OTPService) Verify(ctx context.Context, phone, code string) error {
	if !iranianMobile.MatchString(phone) {
		return ErrInvalidPhone
	}

	latest, err := s.store.Latest(ctx, phone)
	if err != nil {
		return ErrOTPNotFound
	}

	now := s.clock.Now()
	if latest.ConsumedAt != nil {
		return ErrOTPNotFound
	}
	if now.After(latest.ExpiresAt) {
		return ErrOTPExpired
	}
	if latest.Attempts >= otpMaxAttempts {
		return ErrTooManyAttempts
	}
	if latest.CodeHash != hashCode(code) {
		latest.Attempts++
		if saveErr := s.store.Save(ctx, latest); saveErr != nil {
			return fmt.Errorf("record attempt: %w", saveErr)
		}
		return ErrInvalidCode
	}

	consumed := now
	latest.ConsumedAt = &consumed
	if err := s.store.Save(ctx, latest); err != nil {
		return fmt.Errorf("consume code: %w", err)
	}
	return nil
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func randomDigits(n int) (string, error) {
	const digits = "0123456789"
	out := make([]byte, n)
	for i := range out {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		out[i] = digits[idx.Int64()]
	}
	return string(out), nil
}
