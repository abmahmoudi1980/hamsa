package building

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"hamsa/internal/audit"
	"hamsa/internal/auth"
	"hamsa/internal/platform/httpx"
)

// Persian handler messages.
const (
	msgAuthRequired = "احراز هویت لازم است."
	msgManagerOnly  = "این بخش مخصوص مدیر ساختمان است."
	msgIDInvalid    = "شناسه نامعتبر است."
	msgQueryInvalid = "پارامترهای جست‌وجو معتبر نیست."
)

// Register mounts US2 routes under r (expected group: /api/v1 with authMW).
// Every mutating route is audited via the T013 middleware decorator.
func Register(r *gin.RouterGroup, svc *Service, aud *audit.Service) {
	buildings := r.Group("/buildings")
	{
		buildings.GET("", listBuildings(svc))
		buildings.POST("", aud.Middleware("building.create", "building"), createBuilding(svc))

		idB := buildings.Group("/:id")
		{
			idB.GET("", getBuilding(svc))
			idB.PUT("", aud.Middleware("building.update", "building"), updateBuilding(svc))
			idB.DELETE("", aud.Middleware("building.delete", "building"), deleteBuilding(svc))
			idB.GET("/units", listUnits(svc))
			idB.POST("/units", aud.Middleware("unit.create", "unit"), createUnit(svc))
		}
	}

	units := r.Group("/units")
	{
		units.GET("/:id", getUnit(svc))
		units.PUT("/:id", aud.Middleware("unit.update", "unit"), updateUnit(svc))
		units.DELETE("/:id", aud.Middleware("unit.delete", "unit"), deleteUnit(svc))
		units.GET("/:id/history", unitHistory(svc))
	}
}

// requireManager resolves the authenticated user and rejects non-managers.
func requireManager(c *gin.Context) (*auth.User, bool) {
	u := auth.CurrentUser(c)
	if u == nil {
		httpx.WriteError(c, httpx.Unauthorized(msgAuthRequired))
		return nil, false
	}
	if u.Role != auth.RoleManager {
		httpx.WriteError(c, httpx.Forbidden(msgManagerOnly))
		return nil, false
	}
	return u, true
}

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound(msgIDInvalid))
		return uuid.Nil, false
	}
	return id, true
}

// --- buildings ----------------------------------------------------------------

func listBuildings(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		items, err := svc.ListBuildings(c.Request.Context(), mgr)
		if err != nil {
			httpx.WriteError(c, httpx.Internal("خطا در دریافت فهرست ساختمان‌ها."))
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func createBuilding(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		var in BuildingInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest(msgNameRequired))
			return
		}
		b, err := svc.CreateBuilding(c.Request.Context(), mgr, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, b.ID)
		c.Set(audit.CtxAfter, b)
		c.JSON(http.StatusCreated, b)
	}
}

func getBuilding(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		b, err := svc.GetBuilding(c.Request.Context(), mgr, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, b)
	}
}

func updateBuilding(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		var in BuildingInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest(msgNameRequired))
			return
		}
		b, before, err := svc.UpdateBuilding(c.Request.Context(), mgr, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.Set(audit.CtxBefore, before)
		c.Set(audit.CtxAfter, b)
		c.JSON(http.StatusOK, b)
	}
}

func deleteBuilding(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		if err := svc.DeleteBuilding(c.Request.Context(), mgr, id); err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.Status(http.StatusNoContent)
	}
}

// --- units --------------------------------------------------------------------

func listUnits(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c)
		if !ok {
			return
		}
		f, err := unitFilterFromQuery(c)
		if err != nil {
			httpx.WriteError(c, httpx.BadRequest(msgQueryInvalid))
			return
		}
		items, total, err := svc.ListUnits(c.Request.Context(), mgr, buildingID, f)
		if err != nil {
			httpx.WriteError(c, httpx.Internal("خطا در دریافت فهرست واحدها."))
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"items": items, "page": f.Page, "page_size": f.Size, "total": total,
		})
	}
}

// unitFilterFromQuery parses ?q=&block=&floor=&status=&page=&page_size=.
func unitFilterFromQuery(c *gin.Context) (UnitFilter, error) {
	f := UnitFilter{
		Q:      c.Query("q"),
		Block:  c.Query("block"),
		Status: c.Query("status"),
		Page:   atoiDefault(c.Query("page"), 1),
		Size:   atoiDefault(c.Query("page_size"), 20),
	}
	if raw := c.Query("floor"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return f, err
		}
		f.Floor = &v
	}
	return f, nil
}

func atoiDefault(s string, d int) int {
	if s == "" {
		return d
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return d
	}
	return v
}

func createUnit(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c)
		if !ok {
			return
		}
		var in UnitInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest(msgNumberRequired))
			return
		}
		u, err := svc.CreateUnit(c.Request.Context(), mgr, buildingID, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		setUnitAuditCtx(c, u.ID, nil, u)
		c.JSON(http.StatusCreated, u)
	}
}

// setUnitAuditCtx publishes object/before/after for the audit middleware.
func setUnitAuditCtx(c *gin.Context, id uuid.UUID, before, after any) {
	c.Set(audit.CtxObjectID, id)
	if before != nil {
		c.Set(audit.CtxBefore, before)
	}
	if after != nil {
		c.Set(audit.CtxAfter, after)
	}
}

func getUnit(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		u, err := svc.GetUnit(c.Request.Context(), mgr, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, u)
	}
}

func updateUnit(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		var in UnitInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest(msgNumberRequired))
			return
		}
		u, before, err := svc.UpdateUnit(c.Request.Context(), mgr, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		setUnitAuditCtx(c, id, before, u)
		c.JSON(http.StatusOK, u)
	}
}

func deleteUnit(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		if err := svc.DeleteUnit(c.Request.Context(), mgr, id); err != nil {
			writeServiceErr(c, err)
			return
		}
		setUnitAuditCtx(c, id, nil, gin.H{"deleted_at": time.Now().UTC()})
		c.Status(http.StatusNoContent)
	}
}

// unitHistory returns the change history of a unit from the append-only
// audit trail (FR-004), newest first.
func unitHistory(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		// Scope check: 403 when the unit is outside the manager's buildings.
		if _, err := svc.GetUnit(c.Request.Context(), mgr, id); err != nil {
			writeServiceErr(c, err)
			return
		}
		type entry struct {
			ID        int64           `json:"id"`
			UserID    *uuid.UUID      `json:"user_id"`
			Action    string          `json:"action"`
			Before    json.RawMessage `json:"before_value"`
			After     json.RawMessage `json:"after_value"`
			CreatedAt time.Time       `json:"created_at"`
		}
		var items []entry
		err := svc.repo.DB().WithContext(c.Request.Context()).Raw(
			`SELECT id, user_id, action, before_value, after_value, created_at
			   FROM audit_logs
			  WHERE object_type = 'unit' AND object_id = ?
			  ORDER BY created_at DESC, id DESC`, id).Scan(&items).Error
		if err != nil {
			httpx.WriteError(c, httpx.Internal("خطا در دریافت تاریخچه تغییرات."))
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// writeServiceErr maps service-layer errors onto the API envelope; AppErrors
// pass through untouched.
func writeServiceErr(c *gin.Context, err error) {
	if errors.Is(err, ErrNoManager) {
		httpx.WriteError(c, httpx.Unauthorized(msgAuthRequired))
		return
	}
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		httpx.WriteError(c, appErr)
		return
	}
	httpx.WriteError(c, httpx.Internal("خطای داخلی سرور. لطفاً دوباره تلاش کنید."))
}
