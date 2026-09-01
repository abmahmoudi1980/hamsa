package auth

import (
	"context"
	"errors"
	"regexp"
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

type fakeSms struct{ sent []string }

func (f *fakeSms) SendCode(phone, code string) error {
	f.sent = append(f.sent, phone+":"+code)
	return nil
}

// fakeOTPStore keeps records as pointers so in-place mutations (attempts,
// consumed_at) persist without an explicit Save.
type fakeOTPStore struct{ records []*OTPCode }

func (f *fakeOTPStore) Latest(_ context.Context, phone string) (*OTPCode, error) {
	var latest *OTPCode
	for _, r := range f.records {
		if r.Phone == phone && (latest == nil || r.CreatedAt.After(latest.CreatedAt)) {
			latest = r
		}
	}
	if latest == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return latest, nil
}

func (f *fakeOTPStore) Create(_ context.Context, code *OTPCode) error {
	f.records = append(f.records, code)
	return nil
}

func (f *fakeOTPStore) Save(_ context.Context, _ *OTPCode) error { return nil }

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

func newOTPService(t *testing.T, clock *fakeClock, dev bool) (*OTPService, *fakeOTPStore, *fakeSms) {
	t.Helper()
	store := &fakeOTPStore{}
	sender := &fakeSms{}
	return NewOTPService(store, sender, clock, dev), store, sender
}

// --- OTP tests --------------------------------------------------------------

func TestOTPIssue_DevReturnsCodeAndSends(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, store, sender := newOTPService(t, clock, true)

	code, err := svc.Issue(context.Background(), "09121234567")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if len(code) != 6 || !regexp.MustCompile(`^\d{6}$`).MatchString(code) {
		t.Fatalf("expected 6-digit code, got %q", code)
	}
	if len(store.records) != 1 {
		t.Fatalf("expected 1 stored OTP, got %d", len(store.records))
	}
	if store.records[0].CodeHash != hashCode(code) {
		t.Fatalf("stored code not hashed correctly")
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 SMS send, got %d", len(sender.sent))
	}
}

func TestOTPIssue_NonDevHidesCode(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _, _ := newOTPService(t, clock, false)

	code, err := svc.Issue(context.Background(), "09121234567")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if code != "" {
		t.Fatalf("non-dev Issue must not return the code, got %q", code)
	}
}

func TestOTPIssue_InvalidPhone(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _, _ := newOTPService(t, clock, true)

	if _, err := svc.Issue(context.Background(), "12345"); !errors.Is(err, ErrInvalidPhone) {
		t.Fatalf("expected ErrInvalidPhone, got %v", err)
	}
}

func TestOTPIssue_Throttled(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _, _ := newOTPService(t, clock, true)

	if _, err := svc.Issue(context.Background(), "09121234567"); err != nil {
		t.Fatalf("first Issue: %v", err)
	}
	if _, err := svc.Issue(context.Background(), "09121234567"); !errors.Is(err, ErrThrottled) {
		t.Fatalf("expected ErrThrottled, got %v", err)
	}

	clock.advance(otpResendThrottle + time.Second)
	if _, err := svc.Issue(context.Background(), "09121234567"); err != nil {
		t.Fatalf("Issue after throttle window: %v", err)
	}
}

func TestOTPVerify_Success(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, store, _ := newOTPService(t, clock, true)

	code, err := svc.Issue(context.Background(), "09121234567")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if err := svc.Verify(context.Background(), "09121234567", code); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if store.records[0].ConsumedAt == nil {
		t.Fatalf("expected OTP consumed after successful verify")
	}
}

func TestOTPVerify_WrongCode(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, store, _ := newOTPService(t, clock, true)

	if _, err := svc.Issue(context.Background(), "09121234567"); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if err := svc.Verify(context.Background(), "09121234567", "000000"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected ErrInvalidCode, got %v", err)
	}
	if store.records[0].Attempts != 1 {
		t.Fatalf("expected attempts=1, got %d", store.records[0].Attempts)
	}
}

func TestOTPVerify_Expired(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _, _ := newOTPService(t, clock, true)

	code, err := svc.Issue(context.Background(), "09121234567")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	clock.advance(otpValidity + time.Second)
	if err := svc.Verify(context.Background(), "09121234567", code); !errors.Is(err, ErrOTPExpired) {
		t.Fatalf("expected ErrOTPExpired, got %v", err)
	}
}

func TestOTPVerify_TooManyAttempts(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _, _ := newOTPService(t, clock, true)

	code, err := svc.Issue(context.Background(), "09121234567")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	for i := 0; i < otpMaxAttempts; i++ {
		if err := svc.Verify(context.Background(), "09121234567", "000000"); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("attempt %d: expected ErrInvalidCode, got %v", i, err)
		}
	}
	if err := svc.Verify(context.Background(), "09121234567", code); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("expected ErrTooManyAttempts, got %v", err)
	}
}

func TestOTPVerify_NotFound(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	svc, _, _ := newOTPService(t, clock, true)

	if err := svc.Verify(context.Background(), "09121234567", "123456"); !errors.Is(err, ErrOTPNotFound) {
		t.Fatalf("expected ErrOTPNotFound, got %v", err)
	}
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
