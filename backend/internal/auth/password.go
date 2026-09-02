package auth

import (
	"regexp"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"hamsa/internal/platform/httpx"
)

// minimumPasswordLength is the password policy floor (Persian UX: at least 8
// characters). No composition rules — long-but-simple beats short-but-cryptic
// for non-professional users.
const minimumPasswordLength = 8

// ErrWeakPassword, ErrWrongPassword are password domain errors (Persian).
var (
	ErrWeakPassword = httpx.BadRequest("رمز عبور باید حداقل ۸ کاراکتر باشد.")
	ErrWrongLogin   = httpx.Unauthorized("شماره موبایل یا رمز عبور نادرست است.")
	ErrNoPassword   = httpx.Unauthorized("برای این حساب رمز عبور تنظیم نشده است. ابتدا ثبت‌نام کنید.")
)

var hasLetter = regexp.MustCompile(`\p{L}`)
var hasDigit = regexp.MustCompile(`\p{N}`)

// ValidatePassword enforces the password policy.
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < minimumPasswordLength ||
		!hasLetter.MatchString(password) || !hasDigit.MatchString(password) {
		return ErrWeakPassword
	}
	return nil
}

// HashPassword bcrypt-hashes a password (cost 10).
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword reports whether password matches the stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
