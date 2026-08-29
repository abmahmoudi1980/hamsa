package expense

// Handlers: US6 routes (contracts/api.md "Expenses & Financial Report").
// All routes are manager-only (RequireRole) and building-scope-checked in
// the service; create/update/delete are audited via the decorator (FR-038).

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/audit"
	"hamsa/internal/auth"
	"hamsa/internal/platform/httpx"
)

// Register mounts US6 routes on an authenticated group.
func Register(authed *gin.RouterGroup, svc *Service, aud *audit.Service) {
	mgr := authed.Group("", auth.RequireRole(auth.RoleManager))
	mgr.POST("/buildings/:id/expenses", aud.Middleware("expense.record", "expense"), createExpense(svc))
	mgr.GET("/buildings/:id/expenses", listExpenses(svc))
	mgr.GET("/buildings/:id/financial-report", financialReport(svc))
	mgr.GET("/expenses/:id", getExpense(svc))
	mgr.PUT("/expenses/:id", aud.Middleware("expense.update", "expense"), updateExpense(svc))
	mgr.DELETE("/expenses/:id", aud.Middleware("expense.delete", "expense"), deleteExpense(svc))
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

// writeServiceErr maps service errors onto the API envelope (same contract
// as the billing/payment handlers).
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

// --- create ----------------------------------------------------------------------

func createExpense(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in Input
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("داده‌های هزینه نامعتبر است"))
			return
		}
		e, err := svc.Create(c.Request.Context(), u, buildingID, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, e.ID)
		c.Set(audit.CtxAfter, e)
		c.JSON(http.StatusCreated, e)
	}
}

// --- read ------------------------------------------------------------------------

func getExpense(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		e, err := svc.Get(c.Request.Context(), u, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, e)
	}
}

func listExpenses(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c, "id")
		if !ok {
			return
		}
		f := ExpenseFilter{
			Category: c.Query("category"),
			Approval: c.Query("approval"),
			Page:     atoiDefault(c.Query("page"), 1),
			Size:     atoiDefault(c.Query("page_size"), 20),
		}
		if raw := c.Query("from"); raw != "" {
			d, err := time.Parse("2006-01-02", raw)
			if err != nil {
				httpx.WriteError(c, httpx.BadRequest("تاریخ شروع نامعتبر است"))
				return
			}
			f.From = &d
		}
		if raw := c.Query("to"); raw != "" {
			d, err := time.Parse("2006-01-02", raw)
			if err != nil {
				httpx.WriteError(c, httpx.BadRequest("تاریخ پایان نامعتبر است"))
				return
			}
			f.To = &d
		}
		items, total, err := svc.List(c.Request.Context(), u, buildingID, f)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"items": items, "total": total,
			"page": f.Page, "page_size": f.Size,
		})
	}
}

// --- update / delete -------------------------------------------------------------

func updateExpense(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in Input
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("داده‌های هزینه نامعتبر است"))
			return
		}
		before, err := svc.Get(c.Request.Context(), u, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		e, err := svc.Update(c.Request.Context(), u, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, e.ID)
		c.Set(audit.CtxBefore, before)
		c.Set(audit.CtxAfter, e)
		c.JSON(http.StatusOK, e)
	}
}

func deleteExpense(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		before, err := svc.Get(c.Request.Context(), u, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		if err := svc.Delete(c.Request.Context(), u, id); err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.Set(audit.CtxBefore, before)
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

// --- financial report (FR-027) -----------------------------------------------------

func financialReport(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c, "id")
		if !ok {
			return
		}
		rep, err := svc.Report(c.Request.Context(), u, buildingID, c.Query("month"))
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, rep)
	}
}
