package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/platform/httpx"
)

// RefreshToken mirrors the `refresh_tokens` table (migration 0001).
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"column:user_id"`
	FamilyID  uuid.UUID  `gorm:"column:family_id"`
	TokenHash string     `gorm:"column:token_hash"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	RevokedAt *time.Time `gorm:"column:revoked_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

// RefreshStore abstracts refresh_tokens persistence for unit testing.
type RefreshStore interface {
	FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
	Create(ctx context.Context, token *RefreshToken) error
	Save(ctx context.Context, token *RefreshToken) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID, at time.Time) error
}

// GormRefreshStore is the PostgreSQL implementation of RefreshStore.
type GormRefreshStore struct{ DB *gorm.DB }

func (s *GormRefreshStore) FindByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	var t RefreshToken
	if err := s.DB.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *GormRefreshStore) Create(ctx context.Context, token *RefreshToken) error {
	return s.DB.WithContext(ctx).Create(token).Error
}

func (s *GormRefreshStore) Save(ctx context.Context, token *RefreshToken) error {
	return s.DB.WithContext(ctx).Save(token).Error
}

func (s *GormRefreshStore) RevokeFamily(ctx context.Context, familyID uuid.UUID, at time.Time) error {
	return s.DB.WithContext(ctx).Model(&RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", at).Error
}

// Refresh-token domain errors (Persian, per contracts/api.md).
var (
	ErrInvalidToken = httpx.Unauthorized("توکن نامعتبر است.")
	ErrTokenExpired = httpx.Unauthorized("توکن منقضی شده است.")
	ErrTokenReused  = httpx.Unauthorized("توکن قبلاً استفاده شده است.")
)

// TokenService issues and verifies JWT access tokens and rotating refresh
// tokens (research.md R8).
type TokenService struct {
	secret       []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
	refreshStore RefreshStore
	clock        Clock
}

// NewTokenService returns a token service. A nil clock defaults to RealClock;
// refreshStore may be nil only when refresh methods are unused (e.g. access
// parsing in middleware).
func NewTokenService(secret []byte, accessTTL, refreshTTL time.Duration, store RefreshStore, clock Clock) *TokenService {
	if clock == nil {
		clock = RealClock{}
	}
	return &TokenService{
		secret:       secret,
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		refreshStore: store,
		clock:        clock,
	}
}

// IssueAccessToken returns a signed JWT (HS256) whose subject is the user id.
func (s *TokenService) IssueAccessToken(userID uuid.UUID) (token string, expiresAt time.Time, err error) {
	now := s.clock.Now()
	expiresAt = now.Add(s.accessTTL)
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expiresAt, nil
}
func (s *TokenService) ParseAccessToken(token string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return s.secret, nil
	}, jwt.WithTimeFunc(func() time.Time { return s.clock.Now() }))
	if err != nil || !parsed.Valid {
		return uuid.Nil, ErrInvalidToken
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return id, nil
}

// IssueRefreshToken stores a new opaque token and returns its raw value.
func (s *TokenService) IssueRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	raw, err := randomHex(32)
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	now := s.clock.Now()
	rec := &RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		FamilyID:  uuid.New(),
		TokenHash: hashToken(raw),
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
	}
	if err := s.refreshStore.Create(ctx, rec); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}
	return raw, nil
}

// Rotation is the result of a successful refresh.
type Rotation struct {
	UserID       uuid.UUID
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds until the access token expires
}

// RotateRefreshToken verifies a presented refresh token, revokes it, issues a
// successor in the same family, and returns a new access + refresh pair. Reuse
// of an already-used token revokes the whole family (research.md R8).
func (s *TokenService) RotateRefreshToken(ctx context.Context, raw string) (*Rotation, error) {
	rec, err := s.refreshStore.FindByHash(ctx, hashToken(raw))
	if err != nil {
		return nil, ErrInvalidToken
	}

	now := s.clock.Now()
	if rec.RevokedAt != nil {
		// Reuse detected: revoke the entire family.
		if rerr := s.refreshStore.RevokeFamily(ctx, rec.FamilyID, now); rerr != nil {
			return nil, fmt.Errorf("revoke family: %w", rerr)
		}
		return nil, ErrTokenReused
	}
	if now.After(rec.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	newRaw, err := randomHex(32)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	revoked := now
	rec.RevokedAt = &revoked
	if err := s.refreshStore.Save(ctx, rec); err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	next := &RefreshToken{
		ID:        uuid.New(),
		UserID:    rec.UserID,
		FamilyID:  rec.FamilyID,
		TokenHash: hashToken(newRaw),
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
	}
	if err := s.refreshStore.Create(ctx, next); err != nil {
		return nil, fmt.Errorf("store new refresh token: %w", err)
	}

	access, exp, err := s.IssueAccessToken(rec.UserID)
	if err != nil {
		return nil, err
	}

	return &Rotation{
		UserID:       rec.UserID,
		AccessToken:  access,
		RefreshToken: newRaw,
		ExpiresIn:    int64(exp.Sub(s.clock.Now()).Seconds()),
	}, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomHex(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
