package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"hamsa/internal/auth"
	"hamsa/internal/platform/httpx"
)

// Register mounts US9 + US10 routes under v1. The expected outer group is
// the API v1 group with the auth middleware already applied.
//
//   - GET /me/home                       → resident self-service panel (US9 T082)
//   - GET /buildings/:id/dashboard       → manager overview (US10 T086)
//
// Manager scope enforcement is handled inside each service method via the
// shared ScopeResolver; the manager-only path requires the manager role,
// which is enforced by the router group below.
func Register(r *gin.RouterGroup, svc *Service) {
	// Resident: own panel (no role gate — both roles can hit /me/*; the
	// service treats non-residents as having no active units).
	r.GET("/me/home", home(svc))

	// Manager: building dashboard.
	mgr := r.Group("/buildings/:id", auth.RequireRole(auth.RoleManager))
	mgr.GET("/dashboard", buildingDashboard(svc))
}

func home(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := auth.CurrentUser(c)
		if u == nil {
			httpx.WriteError(c, httpx.Unauthorized("برای این عملیات باید وارد شوید"))
			return
		}
		out, err := svc.Home(c.Request.Context(), u)
		if err != nil {
			httpx.WriteError(c, err)
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

func buildingDashboard(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr := auth.CurrentUser(c)
		if mgr == nil {
			httpx.WriteError(c, httpx.Unauthorized("برای این عملیات باید وارد شوید"))
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		out, err := svc.BuildingDashboard(c.Request.Context(), mgr, id)
		if err != nil {
			httpx.WriteError(c, err)
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("شناسه نامعتبر است"))
		return uuid.Nil, false
	}
	return id, true
}
