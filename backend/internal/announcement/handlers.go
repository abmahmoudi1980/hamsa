package announcement

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/auth"
	"hamsa/internal/platform/httpx"
)

// Register mounts US8 routes.
// Expected outer group: v1.Group("", authMW) — all routes require auth.
func Register(r *gin.RouterGroup, svc *Service) {
	// Manager: building-scoped publish & list.
	mgr := r.Group("/buildings/:id", auth.RequireRole(auth.RoleManager))
	{
		mgr.POST("/announcements", createForManager(svc))
		mgr.GET("/announcements", listForManager(svc))
	}

	// Manager: single announcement CRUD (detail/update/delete).
	single := r.Group("/announcements", auth.RequireRole(auth.RoleManager))
	{
		single.GET("/:id", getForManager(svc))
		single.PUT("/:id", updateForManager(svc))
		single.DELETE("/:id", deleteForManager(svc))
	}

	// Resident: own announcements (targeted + publish window) + mark-read.
	me := r.Group("/me")
	{
		me.GET("/announcements", listForResident(svc))
		me.GET("/announcements/:id", getForResident(svc))
	}

	// Mark-read: per contracts/api.md POST /announcements/{id}/read (and /me alias).
	r.POST("/announcements/:id/read", markRead(svc))
	r.POST("/me/announcements/:id/read", markRead(svc))
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
	if err == nil {
		return
	}
	// httpx.AppError passthrough; anything else → 500 via WriteError.
	httpx.WriteError(c, err)
}

func atoiDefault(s string, d int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return d
	}
	return n
}

// --- manager: create ---------------------------------------------------------

func createForManager(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := currentUser(c)
		if !ok {
			return
		}
		bid, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in Input
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("اطلاعات ارسالی نامعتبر است"))
			return
		}
		a, err := svc.Create(c.Request.Context(), mgr, bid, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusCreated, a)
	}
}

func listForManager(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := currentUser(c)
		if !ok {
			return
		}
		bid, ok := parseID(c, "id")
		if !ok {
			return
		}
		page := atoiDefault(c.Query("page"), 1)
		size := atoiDefault(c.Query("page_size"), 20)
		items, total, err := svc.ListForManager(c.Request.Context(), mgr, bid, page, size)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "page_size": size, "total": total})
	}
}

func getForManager(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := currentUser(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		a, err := svc.GetForManager(c.Request.Context(), mgr, id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				httpx.WriteError(c, httpx.NotFound("اطلاعیه یافت نشد"))
				return
			}
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, a)
	}
}

func updateForManager(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := currentUser(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in Input
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("اطلاعات ارسالی نامعتبر است"))
			return
		}
		a, err := svc.Update(c.Request.Context(), mgr, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, a)
	}
}

func deleteForManager(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := currentUser(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		err := svc.Delete(c.Request.Context(), mgr, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// --- resident: list / detail / mark-read ------------------------------------

func listForResident(svc *Service) gin.HandlerFunc {
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
		c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "page_size": size, "total": total})
	}
}

func getForResident(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		a, err := svc.GetForResident(c.Request.Context(), u, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, a)
	}
}

func markRead(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		if err := svc.MarkRead(c.Request.Context(), u, id); err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
