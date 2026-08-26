// Package httpx provides the HTTP foundation for the API: the Gin router
// factory, the JSON error envelope (Persian messages), request-id and
// structured-logging middleware, and response helpers.
package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// FieldError is a single validation failure detail.
type FieldError struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
}

// AppError is the error type carried through handlers; it renders as the
// API error envelope (contracts/api.md):
//
//	{"error":{"code":"...","message":"...","details":[...]}}
type AppError struct {
	Status  int          `json:"-"`
	Code    string       `json:"code"`
	Message string       `json:"message"` // Persian, user-facing
	Details []FieldError `json:"details,omitempty"`
}

// Error implements error.
func (e *AppError) Error() string { return e.Code + ": " + e.Message }

// WithDetail attaches a field/rule validation detail and returns the error.
func (e *AppError) WithDetail(field, rule string) *AppError {
	if e.Details == nil {
		e.Details = []FieldError{}
	}
	e.Details = append(e.Details, FieldError{Field: field, Rule: rule})
	return e
}

// Constructors for the standard error codes (contracts/api.md).

// BadRequest returns 400 VALIDATION_ERROR with a Persian message.
func BadRequest(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: "VALIDATION_ERROR", Message: msg}
}

// Unauthorized returns 401 UNAUTHENTICATED.
func Unauthorized(msg string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Code: "UNAUTHENTICATED", Message: msg}
}

// Forbidden returns 403 FORBIDDEN.
func Forbidden(msg string) *AppError {
	return &AppError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: msg}
}

// NotFound returns 404 NOT_FOUND.
func NotFound(msg string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: msg}
}

// Conflict returns 409 CONFLICT.
func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: "CONFLICT", Message: msg}
}

// RateLimited returns 429 RATE_LIMITED.
func RateLimited(msg string) *AppError {
	return &AppError{Status: http.StatusTooManyRequests, Code: "RATE_LIMITED", Message: msg}
}

// Internal returns 500 INTERNAL.
func Internal(msg string) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Code: "INTERNAL", Message: msg}
}

// ErrInternal is the default sanitized 500 (never leaks internals).
var ErrInternal = Internal("خطای داخلی سرور. لطفاً دوباره تلاش کنید.")

// WriteError renders err as the JSON error envelope, mapping AppError
// directly and sanitizing anything else to a generic 500.
func WriteError(c *gin.Context, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = ErrInternal
	}
	if appErr.Status >= 500 && appErr.Message == ErrInternal.Message {
		// keep 500s sanitized
		appErr = ErrInternal
	}
	c.AbortWithStatusJSON(appErr.Status, gin.H{"error": appErr})
}
