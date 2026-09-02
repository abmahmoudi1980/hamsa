package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/platform/httpx"
)

// Clock abstracts time for testability.
type Clock interface{ Now() time.Time }

// RealClock is the production clock.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

var iranianMobile = regexp.MustCompile(`^09\d{9}$`)

// ErrInvalidPhone is the shared invalid-mobile error (Persian).
var ErrInvalidPhone = httpx.BadRequest("شماره موبایل معتبر نیست.")

// InviteCode lifetimes (one-time manager-issued registration codes).
const (
	inviteValidity     = 7 * 24 * time.Hour
	inviteCodeLength   = 8
	inviteCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O, 1/I: unambiguous when read aloud
)

// Invite domain errors (Persian).
var (
	ErrInviteInvalid       = httpx.Unauthorized("کد دعوت نامعتبر است.")
	ErrInviteExpired       = httpx.Unauthorized("کد دعوت منقضی شده است. از مدیر ساختمان کد جدید بگیرید.")
	ErrInvitePhoneMismatch = httpx.BadRequest("این کد دعوت برای شماره موبایل دیگری صادر شده است.")
)

// InviteCode mirrors the `invite_codes` table (migration 0009).
type InviteCode struct {
	ID         uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	Phone      string     `gorm:"column:phone"`
	CodeHash   string     `gorm:"column:code_hash"`
	CreatedBy  *uuid.UUID `gorm:"column:created_by;type:uuid"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	ConsumedAt *time.Time `gorm:"column:consumed_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
}

func (InviteCode) TableName() string { return "invite_codes" }

// InviteStore abstracts invite_codes persistence.
type InviteStore interface {
	Create(ctx context.Context, code *InviteCode) error
	// Latest returns the newest invite for phone, or gorm.ErrRecordNotFound.
	Latest(ctx context.Context, phone string) (*InviteCode, error)
	// Consume marks the invite consumed (consumed_at = now) in one UPDATE so
	// two concurrent redemptions cannot both succeed.
	Consume(ctx context.Context, id uuid.UUID, at time.Time) error
}

// GormInviteStore is the PostgreSQL implementation of InviteStore.
type GormInviteStore struct{ DB *gorm.DB }

// Create inserts the invite.
func (s *GormInviteStore) Create(ctx context.Context, code *InviteCode) error {
	return s.DB.WithContext(ctx).Create(code).Error
}

// Latest returns the newest invite for phone.
func (s *GormInviteStore) Latest(ctx context.Context, phone string) (*InviteCode, error) {
	var c InviteCode
	if err := s.DB.WithContext(ctx).
		Where("phone = ?", phone).
		Order("created_at DESC").
		First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Consume consumes the invite only if still unconsumed (optimistic guard).
func (s *GormInviteStore) Consume(ctx context.Context, id uuid.UUID, at time.Time) error {
	res := s.DB.WithContext(ctx).Model(&InviteCode{}).
		Where("id = ? AND consumed_at IS NULL", id).
		Update("consumed_at", at)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInviteInvalid // already consumed
	}
	return nil
}

// InviteService issues and redeems one-time invite codes.
type InviteService struct {
	store InviteStore
	clock Clock
}

// NewInviteService returns an invite service. A nil clock defaults to
// RealClock.
func NewInviteService(store InviteStore, clock Clock) *InviteService {
	if clock == nil {
		clock = RealClock{}
	}
	return &InviteService{store: store, clock: clock}
}

// Issue creates a one-time invite for phone and returns its plaintext code.
// The code is shown to the manager exactly once.
func (s *InviteService) Issue(ctx context.Context, phone string, createdBy *uuid.UUID) (string, error) {
	if !iranianMobile.MatchString(phone) {
		return "", ErrInvalidPhone
	}
	code, err := randomInviteCode(inviteCodeLength)
	if err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}
	now := s.clock.Now()
	invite := &InviteCode{
		ID:        uuid.New(),
		Phone:     phone,
		CodeHash:  hashCode(code),
		CreatedBy: createdBy,
		ExpiresAt: now.Add(inviteValidity),
		CreatedAt: now,
	}
	if err := s.store.Create(ctx, invite); err != nil {
		return "", fmt.Errorf("store invite: %w", err)
	}
	return code, nil
}

// Redeem validates phone+code and consumes the invite. A mismatch, expiry,
// or double redemption returns an error; the most recent invite for the phone
// is authoritative.
func (s *InviteService) Redeem(ctx context.Context, phone, code string) error {
	if !iranianMobile.MatchString(phone) {
		return ErrInvalidPhone
	}
	invite, err := s.store.Latest(ctx, phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInviteInvalid
		}
		return fmt.Errorf("load invite: %w", err)
	}
	now := s.clock.Now()
	if now.After(invite.ExpiresAt) {
		return ErrInviteExpired
	}
	// Constant-time compare to keep code checking timing-uniform.
	if subtle.ConstantTimeCompare([]byte(invite.CodeHash), []byte(hashCode(code))) != 1 {
		return ErrInviteInvalid
	}
	return s.store.Consume(ctx, invite.ID, now)
}

// randomInviteCode returns n characters from inviteCodeAlphabet.
func randomInviteCode(n int) (string, error) {
	out := make([]byte, n)
	max := big.NewInt(int64(len(inviteCodeAlphabet)))
	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}

		out[i] = inviteCodeAlphabet[idx.Int64()]
	}
	return string(out), nil
}

// hashCode SHA-256-hashes a one-time code (invites are never stored as
// plaintext).
func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
