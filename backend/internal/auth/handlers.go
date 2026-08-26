package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"hamsa/internal/platform/httpx"
)

// LoginAuditor records security-sensitive login events. Declared here (not as
// a concrete *audit.Service) because the audit package imports auth for
// CurrentUser — an import cycle otherwise. *audit.Service satisfies it.
type LoginAuditor interface {
	Append(ctx context.Context, actorID *uuid.UUID, action, objectType string, objectID *uuid.UUID, before, after any) error
}

// Handler serves the auth endpoints (contracts/api.md "Auth P0-10").
type Handler struct {
	OTP    *OTPService
	Tokens *TokenService
	Users  *Repository
	Scopes *ScopeResolver

	// Auditor appends the user.login audit entry (FR-038) on successful
	// verification; nil disables auditing (tests).
	Auditor LoginAuditor
}

// Register mounts the public auth routes plus GET /me (Bearer-authenticated)
// under r, which must be bound at /auth within /api/v1.
func Register(r *gin.RouterGroup, h *Handler) {
	r.POST("/otp/request", h.requestOTP)
	r.POST("/otp/verify", h.verifyOTP)
	r.POST("/refresh", h.refresh)
	r.POST("/logout", h.logout)

	me := r.Group("", Authenticate(h.Tokens, h.Users))
	me.GET("/me", h.me)
}

type otpRequestReq struct {
	Phone string `json:"phone" binding:"required"`
}

// requestOTP issues a fresh code (60 s resend throttle enforced by OTPService;
// repeat requests inside the window get 429 RATE_LIMITED). In dev mode the
// code is returned in the response so the quickstart works without a real SMS
// provider; production answers 204 with no body.
func (h *Handler) requestOTP(c *gin.Context) {
	var req otpRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, httpx.BadRequest("شماره موبایل الزامی است.").WithDetail("phone", "required"))
		return
	}

	code, err := h.OTP.Issue(c.Request.Context(), req.Phone)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	if code != "" { // dev mode only (OTPService returns "" otherwise)
		c.JSON(http.StatusOK, gin.H{"dev_code": code})
		return
	}
	c.Status(http.StatusNoContent)
}

type verifyReq struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// verifyOTP checks the code and, on success, auto-registers unknown phones
// and returns a token pair plus the user context.
//
// First-user bootstrap: in a fresh deployment (no active users yet) the first
// phone to register is granted the manager role — the product's deployment
// story is "the manager installs the app and signs up"; every later
// self-registration lands as resident until a manager grants buildings
// (user_buildings, US2). Documented per T024.
func (h *Handler) verifyOTP(c *gin.Context) {
	var req verifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, httpx.BadRequest("شماره موبایل و کد تأیید الزامی است."))
		return
	}
	ctx := c.Request.Context()

	if err := h.OTP.Verify(ctx, req.Phone, req.Code); err != nil {
		httpx.WriteError(c, err)
		return
	}

	u, err := h.findOrRegisterUser(ctx, req.Phone)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	access, expiresAt, err := h.Tokens.IssueAccessToken(u.ID)
	if err != nil {
		httpx.WriteError(c, httpx.ErrInternal)
		return
	}
	refresh, err := h.Tokens.IssueRefreshToken(ctx, u.ID)
	if err != nil {
		httpx.WriteError(c, httpx.ErrInternal)
		return
	}

	if h.Auditor != nil {
		if err := h.Auditor.Append(ctx, &u.ID, "user.login", "user", &u.ID, nil, gin.H{"phone": u.Phone, "role": u.Role}); err != nil {
			// Audit failures never block login (FR-038 keeps the trail
			// best-effort at write time; the DB enforces append-only).
			_ = err
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    int64(expiresAt.Sub(h.Tokens.clock.Now()).Seconds()),
		"user":          userPayload(u),
	})
}

// findOrRegisterUser loads the user for phone or creates one; a fresh
// deployment's first user becomes manager (see verifyOTP).
func (h *Handler) findOrRegisterUser(ctx context.Context, phone string) (*User, error) {
	u, err := h.Users.FindByPhone(ctx, phone)
	switch {
	case err == nil:
		return u, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, httpx.Internal("خطا در بازیابی حساب کاربری.")
	}

	count, err := h.Users.CountActive(ctx)
	if err != nil {
		return nil, httpx.Internal("خطا در ثبت‌نام کاربر.")
	}
	role := RoleResident
	if count == 0 {
		role = RoleManager
	}
	u = &User{ID: uuid.New(), Phone: phone, Role: role, IsActive: true}
	if err := h.Users.Create(ctx, u); err != nil {
		return nil, httpx.Internal("خطا در ثبت‌نام کاربر.")
	}
	return u, nil
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *Handler) refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		httpx.WriteError(c, httpx.BadRequest("توکن به‌روزرسانی الزامی است.").WithDetail("refresh_token", "required"))
		return
	}

	rot, err := h.Tokens.RotateRefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  rot.AccessToken,
		"refresh_token": rot.RefreshToken,
		"expires_in":    rot.ExpiresIn,
	})
}

func (h *Handler) logout(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		httpx.WriteError(c, httpx.BadRequest("توکن به‌روزرسانی الزامی است.").WithDetail("refresh_token", "required"))
		return
	}

	if err := h.Tokens.RevokeFamilyByToken(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.WriteError(c, httpx.Internal("خطا در خروج از حساب."))
		return
	}
	c.Status(http.StatusNoContent)
}

// me returns the current identity plus its server-resolved scope: manager →
// permitted building ids (user_buildings), resident → occupied unit ids
// (occupancies). The scope tables arrive with the US2/US3 migrations
// (migration 0002/0003); until then the queries hit undefined tables, which
// resolves to empty lists rather than an error.
func (h *Handler) me(c *gin.Context) {
	u := CurrentUser(c)
	if u == nil {
		httpx.WriteError(c, httpx.Unauthorized("احراز هویت لازم است."))
		return
	}
	ctx := c.Request.Context()

	resp := gin.H{"id": u.ID, "phone": u.Phone, "name": u.Name, "role": u.Role}

	switch u.Role {
	case RoleManager:
		buildings, err := h.Scopes.ManagerBuildingIDs(ctx, u.ID)
		if err != nil && !isUndefinedTable(err) {
			httpx.WriteError(c, httpx.Internal("خطا در دریافت ساختمان‌های مجاز."))
			return
		}
		resp["buildings"] = uuids(buildings)
	case RoleResident:
		units, err := h.Scopes.ResidentUnitIDs(ctx, u.Phone)
		if err != nil && !isUndefinedTable(err) {
			httpx.WriteError(c, httpx.Internal("خطا در دریافت واحدهای سکونت."))
			return
		}
		resp["units"] = uuids(units)
	}

	c.JSON(http.StatusOK, resp)
}

func userPayload(u *User) gin.H {
	return gin.H{"id": u.ID, "name": u.Name, "role": u.Role}
}

func uuids(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

// isUndefinedTable reports PostgreSQL SQLSTATE 42P01 (relation does not
// exist) — used for scope tables that ship with later migrations.
func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}
