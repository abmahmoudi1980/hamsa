package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- test fakes -------------------------------------------------------------

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }
func (c *fakeClock) advance(d time.Duration) {
	c.t = c.t.Add(d)
}

type fakeRefreshStore struct{ records []*RefreshToken }

func (f *fakeRefreshStore) FindByHash(_ context.Context, hash string) (*RefreshToken, error) {
	for _, r := range f.records {
		if r.TokenHash == hash {
			return r, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRefreshStore) Create(_ context.Context, t *RefreshToken) error {
	f.records = append(f.records, t)
	return nil
}

func (f *fakeRefreshStore) Save(_ context.Context, _ *RefreshToken) error { return nil }

func (f *fakeRefreshStore) RevokeFamily(_ context.Context, familyID uuid.UUID, at time.Time) error {
	for _, r := range f.records {
		if r.FamilyID == familyID && r.RevokedAt == nil {
			r.RevokedAt = &at
		}
	}
	return nil
}

// --- JWT tests --------------------------------------------------------------

func newTokenService(clock *fakeClock, accessTTL, refreshTTL time.Duration) (*TokenService, *fakeRefreshStore) {
	store := &fakeRefreshStore{}
	svc := NewTokenService([]byte("test-secret"), accessTTL, refreshTTL, store, clock)
	return svc, store
}

func TestAccessToken_Roundtrip(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _ := newTokenService(clock, 15*time.Minute, 30*24*time.Hour)

	id := uuid.New()
	token, _, err := svc.IssueAccessToken(id)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	got, err := svc.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if got != id {
		t.Fatalf("expected %s, got %s", id, got)
	}
}

func TestAccessToken_Expired(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _ := newTokenService(clock, time.Minute, 30*24*time.Hour)

	token, _, err := svc.IssueAccessToken(uuid.New())
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	clock.advance(2 * time.Minute)
	if _, err := svc.ParseAccessToken(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAccessToken_WrongSecret(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _ := newTokenService(clock, 15*time.Minute, 30*24*time.Hour)
	other := NewTokenService([]byte("other-secret"), 15*time.Minute, 30*24*time.Hour, &fakeRefreshStore{}, clock)

	token, _, err := svc.IssueAccessToken(uuid.New())
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	if _, err := other.ParseAccessToken(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

// --- refresh rotation tests -------------------------------------------------

func TestRefreshToken_IssueAndRotate(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, store := newTokenService(clock, 15*time.Minute, 30*24*time.Hour)

	userID := uuid.New()
	raw, err := svc.IssueRefreshToken(context.Background(), userID)
	if err != nil {
		t.Fatalf("IssueRefreshToken: %v", err)
	}
	if len(store.records) != 1 {
		t.Fatalf("expected 1 stored token, got %d", len(store.records))
	}

	rot, err := svc.RotateRefreshToken(context.Background(), raw)
	if err != nil {
		t.Fatalf("RotateRefreshToken: %v", err)
	}
	if rot.UserID != userID {
		t.Fatalf("expected userID %s, got %s", userID, rot.UserID)
	}
	if rot.RefreshToken == "" || rot.AccessToken == "" {
		t.Fatalf("rotation must return new access + refresh tokens")
	}
	if len(store.records) != 2 {
		t.Fatalf("expected 2 stored tokens after rotation, got %d", len(store.records))
	}
	// Old token revoked, new token active, same family.
	old, _ := store.FindByHash(context.Background(), hashToken(raw))
	if old.RevokedAt == nil {
		t.Fatalf("old token must be revoked after rotation")
	}
	newRec, _ := store.FindByHash(context.Background(), hashToken(rot.RefreshToken))
	if newRec.RevokedAt != nil {
		t.Fatalf("new token must be active")
	}
	if old.FamilyID != newRec.FamilyID {
		t.Fatalf("rotation must preserve the family")
	}
}

func TestRefreshToken_ReuseRevokesFamily(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, store := newTokenService(clock, 15*time.Minute, 30*24*time.Hour)

	raw, err := svc.IssueRefreshToken(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("IssueRefreshToken: %v", err)
	}
	rot1, err := svc.RotateRefreshToken(context.Background(), raw)
	if err != nil {
		t.Fatalf("first rotate: %v", err)
	}
	if _, err := svc.RotateRefreshToken(context.Background(), rot1.RefreshToken); err != nil {
		t.Fatalf("second rotate: %v", err)
	}

	// Reusing rot1's (already-consumed) token must revoke the whole family.
	if _, err := svc.RotateRefreshToken(context.Background(), rot1.RefreshToken); !errors.Is(err, ErrTokenReused) {
		t.Fatalf("expected ErrTokenReused, got %v", err)
	}
	for _, r := range store.records {
		if r.RevokedAt == nil {
			t.Fatalf("all tokens in family must be revoked after reuse")
		}
	}
}

func TestRefreshToken_Expired(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _ := newTokenService(clock, 15*time.Minute, time.Hour)

	raw, err := svc.IssueRefreshToken(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("IssueRefreshToken: %v", err)
	}
	clock.advance(2 * time.Hour)
	if _, err := svc.RotateRefreshToken(context.Background(), raw); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _ := newTokenService(clock, 15*time.Minute, time.Hour)

	if _, err := svc.RotateRefreshToken(context.Background(), "does-not-exist"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
