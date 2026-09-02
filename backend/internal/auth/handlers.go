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

// Handler serves the auth endpoints (phone+password login; one-time
// manager-issued invite codes for registration and password reset).
type Handler struct {
	Tokens  *TokenService
	Users   *Repository
	Scopes  *ScopeResolver
	Invites *InviteService

	// Auditor appends the user.login audit entry (FR-038) on successful
	// login; nil disables auditing (tests).
	Auditor LoginAuditor
}

// Register mounts the public auth routes plus the authenticated /auth routes
// under r, which must be bound at /auth within /api/v1.
func Register(r *gin.RouterGroup, h *Handler) {
	r.POST("/setup", h.setup)
	r.POST("/register", h.register)
	r.POST("/login", h.login)
	r.POST("/refresh", h.refresh)
	r.POST("/logout", h.logout)

	authed := r.Group("", Authenticate(h.Tokens, h.Users))
	authed.GET("/me", h.me)
	authed.POST("/password", h.changePassword)
	authed.POST("/invites", RequireRole(RoleManager, RoleSuperAdmin), h.createInvite)
}

type credentialsReq struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name"`
}

// setup bootstraps a fresh deployment: when no active users exist, the first
// caller becomes the superadmin (002-multi-manager-support) — the single,
// permanent platform account whose privilege is issuing manager invites.
// Once any user exists the endpoint is closed (409) — every later account
// arrives through /register with an invite code.
func (h *Handler) setup(c *gin.Context) {
	var req credentialsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, httpx.BadRequest("شماره موبایل و رمز عبور الزامی است."))
		return
	}
	ctx := c.Request.Context()

	if !iranianMobile.MatchString(req.Phone) {
		httpx.WriteError(c, ErrInvalidPhone)
		return
	}
	if err := ValidatePassword(req.Password); err != nil {
		httpx.WriteError(c, err)
		return
	}

	count, err := h.Users.CountActive(ctx)
	if err != nil {
		httpx.WriteError(c, httpx.Internal("خطا در بررسی حساب‌های موجود."))
		return
	}
	if count > 0 {
		httpx.WriteError(c, httpx.Conflict("حساب مدیریتی قبلاً ساخته شده است. برای ورود از صفحه ورود استفاده کنید."))
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(c, httpx.ErrInternal)
		return
	}
	u := &User{ID: uuid.New(), Phone: req.Phone, Role: RoleSuperAdmin, Name: req.Name, IsActive: true, PasswordHash: &hash}
	if err := h.Users.Create(ctx, u); err != nil {
		httpx.WriteError(c, httpx.Internal("خطا در ساخت حساب مدیریتی."))
		return
	}
	h.respondSession(c, u)
}

// register redeems a one-time invite code. Unknown phones register with the
// invite's role (resident | manager); an existing user with the same phone
// gets their password reset ONLY — never a role change, in either direction
// (the invite is the trusted channel that proves control of the phone, so it
// doubles as password recovery).
type registerReq struct {
	credentialsReq
	Code string `json:"code" binding:"required"`
}

func (h *Handler) register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, httpx.BadRequest("شماره موبایل، کد دعوت و رمز عبور الزامی است."))
		return
	}
	ctx := c.Request.Context()

	if !iranianMobile.MatchString(req.Phone) {
		httpx.WriteError(c, ErrInvalidPhone)
		return
	}
	if err := ValidatePassword(req.Password); err != nil {
		httpx.WriteError(c, err)
		return
	}
	role, err := h.Invites.Redeem(ctx, req.Phone, req.Code)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(c, httpx.ErrInternal)
		return
	}

	u, err := h.Users.FindByPhone(ctx, req.Phone)
	switch {
	case err == nil:
		u.PasswordHash = &hash
		if req.Name != "" {
			u.Name = req.Name
		}
		if err := h.Users.UpdatePassword(ctx, u); err != nil {
			httpx.WriteError(c, httpx.Internal("خطا در ذخیره رمز عبور."))
			return
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		u = &User{ID: uuid.New(), Phone: req.Phone, Role: role, Name: req.Name, IsActive: true, PasswordHash: &hash}
		if err := h.Users.Create(ctx, u); err != nil {
			httpx.WriteError(c, httpx.Internal("خطا در ثبت‌نام کاربر."))
			return
		}
	default:
		httpx.WriteError(c, httpx.Internal("خطا در بازیابی حساب کاربری."))
		return
	}

	h.respondSession(c, u)
}

// login authenticates phone+password.
func (h *Handler) login(c *gin.Context) {
	var req credentialsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, httpx.BadRequest("شماره موبایل و رمز عبور الزامی است."))
		return
	}
	ctx := c.Request.Context()

	if !iranianMobile.MatchString(req.Phone) {
		httpx.WriteError(c, ErrWrongLogin)
		return
	}

	u, err := h.Users.FindByPhone(ctx, req.Phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.WriteError(c, ErrWrongLogin) // same answer for unknown phone: no account enumeration
			return
		}
		httpx.WriteError(c, httpx.Internal("خطا در بازیابی حساب کاربری."))
		return
	}
	if u.PasswordHash == nil || !CheckPassword(*u.PasswordHash, req.Password) {
		httpx.WriteError(c, ErrWrongLogin)
		return
	}

	h.respondSession(c, u)
}

type changePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// changePassword sets a new password after verifying the current one.
func (h *Handler) changePassword(c *gin.Context) {
	u := CurrentUser(c)
	if u == nil {
		httpx.WriteError(c, httpx.Unauthorized("احراز هویت لازم است."))
		return
	}
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, httpx.BadRequest("رمز عبور فعلی و رمز جدید الزامی است."))
		return
	}
	if err := ValidatePassword(req.NewPassword); err != nil {
		httpx.WriteError(c, err)
		return
	}
	if u.PasswordHash == nil || !CheckPassword(*u.PasswordHash, req.CurrentPassword) {
		httpx.WriteError(c, ErrWrongLogin)
		return
	}

	hash, err := HashPassword(req.NewPassword)
	if err != nil {
		httpx.WriteError(c, httpx.ErrInternal)
		return
	}
	u.PasswordHash = &hash
	if err := h.Users.UpdatePassword(c.Request.Context(), u); err != nil {
		httpx.WriteError(c, httpx.Internal("خطا در ذخیره رمز عبور."))
		return
	}
	c.Status(http.StatusNoContent)
}

type createInviteReq struct {
	Phone string `json:"phone" binding:"required"`
	Role  string `json:"role"`
}

// createInvite issues a one-time invite code for a phone (manager or
// superadmin). The plaintext code is returned exactly once; the issuer passes
// it to the invitee out-of-band. role defaults to resident; manager invites
// grant nothing by themselves — the registrant only becomes a *building*
// manager once an existing manager grants them a building.
func (h *Handler) createInvite(c *gin.Context) {
	caller := CurrentUser(c)
	var req createInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, httpx.BadRequest("شماره موبایل الزامی است."))
		return
	}

	role := req.Role
	if role == "" {
		role = RoleResident
	}
	if role != RoleResident && role != RoleManager {
		httpx.WriteError(c, httpx.BadRequest("نقش دعوت نامعتبر است."))
		return
	}

	ctx := c.Request.Context()
	code, err := h.Invites.Issue(ctx, req.Phone, role, &caller.ID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	if h.Auditor != nil {
		_ = h.Auditor.Append(ctx, &caller.ID, "invite.issued", "invite", nil, nil,
			map[string]any{"phone": req.Phone, "role": role})
	}
	c.JSON(http.StatusCreated, gin.H{"code": code, "role": role, "expires_in_days": 7})
}

// respondSession issues a token pair and answers the standard session body.
func (h *Handler) respondSession(c *gin.Context, u *User) {
	ctx := c.Request.Context()

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
		_ = h.Auditor.Append(ctx, &u.ID, "user.login", "user", &u.ID, nil, gin.H{"phone": u.Phone, "role": u.Role})
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    int64(expiresAt.Sub(h.Tokens.clock.Now()).Seconds()),
		"user":          userPayload(u),
	})
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
// (occupancies).
func (h *Handler) me(c *gin.Context) {
	u := CurrentUser(c)
	if u == nil {
		httpx.WriteError(c, httpx.Unauthorized("احراز هویت لازم است."))
		return
	}
	ctx := c.Request.Context()

	resp := gin.H{"id": u.ID, "phone": u.Phone, "name": u.Name, "role": u.Role}

	switch u.Role {
	case RoleSuperAdmin:
		// The superadmin governs no buildings (research R11): empty, never
		// absent — the client renders a definite "nothing here" state.
		resp["buildings"] = []uuid.UUID{}
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
