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

// --- password policy -------------------------------------------------------

func TestValidatePassword(t *testing.T) {
	valid := []string{"hamsha-1234", "xxxxxx1x", "رمز12345"}
	for _, p := range valid {
		if err := ValidatePassword(p); err != nil {
			t.Errorf("ValidatePassword(%q) = %v, want nil", p, err)
		}
	}
	invalid := []string{"", "short1a", "12345678", "abcdefgh"}
	for _, p := range invalid {
		if err := ValidatePassword(p); !errors.Is(err, ErrWeakPassword) {
			t.Errorf("ValidatePassword(%q) = %v, want ErrWeakPassword", p, err)
		}
	}
}

func TestHashPasswordRoundtrip(t *testing.T) {
	hash, err := HashPassword("hamsha-1234")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "hamsha-1234" || len(hash) < 50 {
		t.Fatalf("hash looks wrong: %q", hash)
	}
	if !CheckPassword(hash, "hamsha-1234") {
		t.Fatal("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password accepted")
	}
}

// --- invite service --------------------------------------------------------

// fakeInviteStore keeps records as pointers so Consume mutations persist.
type fakeInviteStore struct{ records []*InviteCode }

func (f *fakeInviteStore) Create(_ context.Context, code *InviteCode) error {
	f.records = append(f.records, code)
	return nil
}

func (f *fakeInviteStore) Latest(_ context.Context, phone string) (*InviteCode, error) {
	var latest *InviteCode
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

func (f *fakeInviteStore) Consume(_ context.Context, id uuid.UUID, at time.Time) error {
	for _, r := range f.records {
		if r.ID == id && r.ConsumedAt == nil {
			r.ConsumedAt = &at
			return nil
		}
	}
	return ErrInviteInvalid
}

func newInviteService(t *testing.T) (*InviteService, *fakeInviteStore, *fakeClock) {
	t.Helper()
	store := &fakeInviteStore{}
	clock := &fakeClock{t: time.Now()}
	return NewInviteService(store, clock), store, clock
}

func TestInviteIssueAndRedeem(t *testing.T) {
	svc, store, clock := newInviteService(t)
	ctx := context.Background()
	const phone = "09121234567"

	code, err := svc.Issue(ctx, phone, nil)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if len(code) != inviteCodeLength {
		t.Fatalf("code length %d want %d", len(code), inviteCodeLength)
	}
	if matched, _ := regexp.MatchString(`^["`+inviteCodeAlphabet+`"]+$`, code); !matched {
		t.Fatalf("code %q outside alphabet", code)
	}
	if store.records[0].CodeHash == code {
		t.Fatal("invite stored as plaintext")
	}

	if err := svc.Redeem(ctx, phone, code); err != nil {
		t.Fatalf("Redeem: %v", err)
	}

	// Double redemption fails.
	if err := svc.Redeem(ctx, phone, code); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("re-redeem: got %v want ErrInviteInvalid", err)
	}
	_ = clock
}

func TestInviteRedeem_WrongCodeAndPhone(t *testing.T) {
	svc, _, _ := newInviteService(t)
	ctx := context.Background()

	if _, err := svc.Issue(ctx, "09121234567", nil); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if err := svc.Redeem(ctx, "09121234567", "ZZZZZZZZ"); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("wrong code: got %v want ErrInviteInvalid", err)
	}
	if err := svc.Redeem(ctx, "09998887766", "AAAAAAAA"); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("unknown phone: got %v want ErrInviteInvalid", err)
	}
	if err := svc.Redeem(ctx, "12345", "AAAAAAAA"); !errors.Is(err, ErrInvalidPhone) {
		t.Fatalf("invalid phone: got %v want ErrInvalidPhone", err)
	}
}

func TestInviteRedeem_Expired(t *testing.T) {
	svc, _, clock := newInviteService(t)
	ctx := context.Background()

	code, err := svc.Issue(ctx, "09121234567", nil)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	clock.advance(inviteValidity + time.Minute)
	if err := svc.Redeem(ctx, "09121234567", code); !errors.Is(err, ErrInviteExpired) {
		t.Fatalf("expired redeem: got %v want ErrInviteExpired", err)
	}
}
