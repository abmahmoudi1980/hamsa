package maintenance

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/audit"
	"hamsa/internal/auth"
	"hamsa/internal/platform/httpx"
)

// Register mounts US7 routes on an authenticated group (caller: v1.Group("", authMW)).
func Register(r *gin.RouterGroup, svc *Service, aud *audit.Service) {
	// Resident: submit & own list/detail.
	me := r.Group("/me")
	{
		me.POST("/maintenance-requests", submitResident(svc))
		me.GET("/maintenance-requests", listResident(svc))
		me.GET("/maintenance-requests/:id", getResident(svc))
	}

	// Manager: per-building list and single-item patch/detail.
	// Building list requires manager role; the service re-checks user_buildings.
	buildings := r.Group("/buildings/:id")
	{
		buildings.GET("/maintenance-requests", listForManager(svc))
	}

	// Single request — manager-only.
	r.PATCH("/maintenance-requests/:id", patchForManager(svc))
	r.GET("/maintenance-requests/:id", getForManager(svc))

	// Keep the aud param referenced for callers that pass it; if a future
	// revision wants decorator audits for maintenance, it can be wired here.
	_ = aud
}

func currentUser(c *gin.Context) (*auth.User, bool) {
	u := auth.CurrentUser(c)
	if u == nil {
		httpx.WriteError(c, httpx.Unauthorized("برای این عملیات باید وارد شوید"))
		return nil, false
	}
	return u, true
}

func parseID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("شناسه نامعتبر است"))
		return uuid.Nil, false
	}
	return id, true
}

func writeServiceErr(c *gin.Context, err error) {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		httpx.WriteError(c, appErr)
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		httpx.WriteError(c, httpx.NotFound("یافت نشد"))
		return
	}
	httpx.WriteError(c, httpx.Internal("خطای داخلی سرور. لطفاً دوباره تلاش کنید."))
}

func atoiDefault(s string, d int) int {
	if s == "" {
		return d
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return d
	}
	return n
}

// --- resident: submit ------------------------------------------------------

func submitResident(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		var in SubmitInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("داده‌های درخواست نگهداری نامعتبر است"))
			return
		}
		m, err := svc.Submit(c.Request.Context(), u, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		// Audit context for optional decorator (if caller adds it).
		c.Set(audit.CtxObjectID, m.ID)
		c.Set(audit.CtxAfter, m)
		c.JSON(http.StatusCreated, m)
	}
}

// --- resident: own list ----------------------------------------------------

func listResident(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		page := atoiDefault(c.Query("page"), 1)
		size := atoiDefault(c.Query("page_size"), 20)
		items, total, err := svc.ListForResident(c.Request.Context(), u, page, size)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		if items == nil {
			items = []MaintenanceRequest{}
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": size})
	}
}

func getResident(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		m, err := svc.GetForResident(c.Request.Context(), u, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, m)
	}
}

// --- manager: building list ------------------------------------------------

func listForManager(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := currentUser(c)
		if !ok {
			return
		}
		if mgr.Role != auth.RoleManager {
			httpx.WriteError(c, httpx.Forbidden("این عملیات مخصوص مدیر است"))
			return
		}
		buildingID, ok := parseID(c, "id")
		if !ok {
			return
		}
		f := Filter{
			Status:   c.Query("status"),
			Priority: c.Query("priority"),
			Category: c.Query("category"),
			Page:     atoiDefault(c.Query("page"), 1),
			Size:     atoiDefault(c.Query("page_size"), 20),
		}
		items, total, err := svc.ListForManager(c.Request.Context(), mgr, buildingID, f)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		if items == nil {
			items = []MaintenanceRequest{}
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": f.Page, "page_size": f.Size})
	}
}

// --- manager: single patch / get ------------------------------------------

func patchForManager(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := currentUser(c)
		if !ok {
			return
		}
		if mgr.Role != auth.RoleManager {
			httpx.WriteError(c, httpx.Forbidden("این عملیات مخصوص مدیر است"))
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in UpdateInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("داده‌های به‌روزرسانی نامعتبر است"))
			return
		}
		// Normalize empty-string status (client sent "" or whitespace) to nil — no transition.
		if in.Status != nil && len(*in.Status) == 0 {
			in.Status = nil
		}
		m, before, err := svc.Update(c.Request.Context(), mgr, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, m.ID)
		c.Set(audit.CtxBefore, before)
		c.Set(audit.CtxAfter, m)
		c.JSON(http.StatusOK, m)
	}
}

func getForManager(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := currentUser(c)
		if !ok {
			return
		}
		if mgr.Role != auth.RoleManager {
			httpx.WriteError(c, httpx.Forbidden("این عملیات مخصوص مدیر است"))
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		m, err := svc.GetForManager(c.Request.Context(), mgr, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, m)
	}
}
