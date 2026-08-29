package billing

// T049 — HTTP handlers for US4 (contracts/api.md "Billing — periods, cost
// items, calculation" + "Invoices"): periods CRUD (edit only in draft), cost
// items CRUD (draft only), calculate, preview, reopen/issue/close, invoice
// list/detail, cancel/adjustments, and resident /me/invoices. Persian
// messages, 409 on illegal state transitions.

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/audit"
	"hamsa/internal/auth"
	"hamsa/internal/platform/httpx"
)

// Persian handler messages.
const (
	msgAuthRequired = "ابتدا وارد شوید"
	msgManagerOnly  = "این عملیات مخصوص مدیر است"
	msgIDInvalid    = "شناسه نامعتبر است"
)

// Register mounts US4 routes under r (expected group: /api/v1 with authMW —
// role is enforced per route: manager routes reject residents with 403;
// /me/* routes scope to the resident's own units).
func Register(r *gin.RouterGroup, calc *CalcService, periods *PeriodService, aud *audit.Service, scopes *auth.ScopeResolver) {
	// --- periods (manager) -------------------------------------------------------
	buildings := r.Group("/buildings/:id")
	{
		buildings.GET("/periods", listPeriods(periods))
		buildings.POST("/periods", aud.Middleware("period.create", "billing_period"), createPeriod(periods))
		buildings.GET("/invoices", listBuildingInvoices(periods))
	}

	periodsG := r.Group("/periods/:id")
	{
		periodsG.GET("", getPeriod(periods))
		periodsG.PATCH("", aud.Middleware("period.update", "billing_period"), updatePeriod(periods))
		periodsG.GET("/cost-items", listCostItems(periods))
		periodsG.POST("/cost-items", aud.Middleware("cost_item.create", "cost_item"), addCostItem(periods))
		periodsG.POST("/calculate", aud.Middleware("period.calculate", "billing_period"), calculate(calc))
		periodsG.GET("/preview", preview(calc))
		periodsG.POST("/reopen", reopen(periods))
		periodsG.POST("/issue", issue(periods))
		periodsG.POST("/close", close(periods))
	}

	// --- invoices ------------------------------------------------------------------

	costItems := r.Group("/cost-items/:id")
	{
		costItems.PUT("", aud.Middleware("cost_item.update", "cost_item"), updateCostItem(periods))
		costItems.DELETE("", aud.Middleware("cost_item.delete", "cost_item"), deleteCostItem(periods))
	}

	r.GET("/invoices/:id", getInvoice(periods, scopes))
	r.POST("/invoices/:id/cancel", aud.Middleware("invoice.cancel", "invoice"), cancelInvoice(periods))
	r.POST("/invoices/:id/adjustments", aud.Middleware("invoice.adjust", "invoice"), addAdjustment(periods))

	// --- resident scope --------------------------------------------------------------
	me := r.Group("/me")
	{
		me.GET("/invoices", myInvoices(periods, scopes))
		me.GET("/invoices/:id", myInvoice(periods, scopes))
	}
}

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

func parseID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound(msgIDInvalid))
		return uuid.Nil, false
	}
	return id, true
}

// writeServiceErr maps service errors onto the API envelope; AppErrors pass
// through, not-found maps to 404, anything else sanitizes to 500.
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

// --- periods -----------------------------------------------------------------------

func listPeriods(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c, "id")
		if !ok {
			return
		}
		items, err := svc.ListPeriods(c.Request.Context(), mgr, buildingID)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func createPeriod(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in PeriodInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("عنوان و تاریخ‌های دوره الزامی است"))
			return
		}
		p, err := svc.CreatePeriod(c.Request.Context(), mgr, buildingID, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, p.ID)
		c.Set(audit.CtxAfter, p)
		c.JSON(http.StatusCreated, p)
	}
}

func getPeriod(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		p, err := svc.GetPeriod(c.Request.Context(), mgr, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

func updatePeriod(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in PeriodInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("عنوان و تاریخ‌های دوره الزامی است"))
			return
		}
		p, before, err := svc.UpdatePeriod(c.Request.Context(), mgr, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.Set(audit.CtxBefore, before)
		c.Set(audit.CtxAfter, p)
		c.JSON(http.StatusOK, p)
	}
}

// --- cost items ----------------------------------------------------------------------

func listCostItems(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		periodID, ok := parseID(c, "id")
		if !ok {
			return
		}
		if _, err := svc.GetPeriod(c.Request.Context(), mgr, periodID); err != nil {
			writeServiceErr(c, err)
			return
		}
		items, err := svc.repo.ListCostItems(c.Request.Context(), periodID)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func addCostItem(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		periodID, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in CostItemInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("عنوان قلم هزینه و روش محاسبه الزامی است"))
			return
		}
		ci, err := svc.AddCostItem(c.Request.Context(), mgr, periodID, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, ci.ID)
		c.Set(audit.CtxAfter, ci)
		c.JSON(http.StatusCreated, ci)
	}
}

func updateCostItem(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in CostItemInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("عنوان قلم هزینه و روش محاسبه الزامی است"))
			return
		}
		ci, before, err := svc.UpdateCostItem(c.Request.Context(), mgr, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.Set(audit.CtxBefore, before)
		c.Set(audit.CtxAfter, ci)
		c.JSON(http.StatusOK, ci)
	}
}

func deleteCostItem(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		if err := svc.DeleteCostItem(c.Request.Context(), mgr, id); err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.Status(http.StatusNoContent)
	}
}

// --- calculation -----------------------------------------------------------------------

func calculate(svc *CalcService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		preview, err := svc.Calculate(c.Request.Context(), mgr, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.Set(audit.CtxAfter, map[string]any{"action": "calculate", "reconciled": preview.Reconciled})
		c.JSON(http.StatusOK, preview)
	}
}

func preview(svc *CalcService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		preview, err := svc.Preview(c.Request.Context(), mgr, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, preview)
	}
}

func reopen(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		p, err := svc.Reopen(c.Request.Context(), mgr, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

func issue(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		issued, err := svc.Issue(c.Request.Context(), mgr, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.Set(audit.CtxAfter, map[string]any{"issued_count": len(issued)})
		c.JSON(http.StatusOK, gin.H{"items": issued, "total": len(issued)})
	}
}

func close(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		p, err := svc.Close(c.Request.Context(), mgr, id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

// --- invoices --------------------------------------------------------------------------

func listBuildingInvoices(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c, "id")
		if !ok {
			return
		}
		f := InvoiceFilter{BuildingID: buildingID}
		if s := c.Query("period_id"); s != "" {
			if pid, err := uuid.Parse(s); err == nil {
				f.PeriodID = &pid
			}
		}
		if s := c.Query("unit_id"); s != "" {
			if uid, err := uuid.Parse(s); err == nil {
				f.UnitID = &uid
			}
		}
		f.Status = c.Query("status")
		f.Page = atoiDefault(c.Query("page"), 1)
		f.PageSize = atoiDefault(c.Query("page_size"), 20)
		items, total, err := svc.ListInvoices(c.Request.Context(), mgr, f)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "page": f.Page, "page_size": f.PageSize, "total": total})
	}
}

func getInvoice(svc *PeriodService, scopes *auth.ScopeResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := auth.CurrentUser(c)
		if u == nil {
			httpx.WriteError(c, httpx.Unauthorized(msgAuthRequired))
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		inv, err := svc.repo.GetInvoice(c.Request.Context(), id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		// Object-level authorization (contracts/api.md behavior 2): managers
		// need the building scope; residents need to own the invoiced unit.
		switch {
		case u.Role == auth.RoleManager:
			ok, err := svc.repo.IsManagerOf(c.Request.Context(), u.ID, inv.BuildingID)
			if err != nil {
				writeServiceErr(c, err)
				return
			}
			if !ok {
				httpx.WriteError(c, httpx.Forbidden(msgNoBuildingAccess))
				return
			}
		default:
			units, err := scopes.ResidentUnitIDs(c.Request.Context(), u.Phone)
			if err != nil {
				writeServiceErr(c, err)
				return
			}
			allowed := false
			for _, uid := range units {
				if uid == inv.UnitID {
					allowed = true
					break
				}
			}
			if !allowed {
				httpx.WriteError(c, httpx.Forbidden("شما به این صورتحساب دسترسی ندارید"))
				return
			}
		}
		c.JSON(http.StatusOK, inv)
	}
}

func cancelInvoice(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in CancelInvoiceInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("دلیل ابطال الزامی است"))
			return
		}
		inv, err := svc.CancelInvoice(c.Request.Context(), mgr, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.JSON(http.StatusOK, inv)
	}
}

func addAdjustment(svc *PeriodService) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in AdjustmentInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("نوع، مبلغ و دلیل اصلاحیه الزامی است"))
			return
		}
		adj, err := svc.AddAdjustment(c.Request.Context(), mgr, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.Set(audit.CtxAfter, adj)
		c.JSON(http.StatusCreated, adj)
	}
}

// --- resident scope ---------------------------------------------------------------------

func myInvoices(svc *PeriodService, scopes *auth.ScopeResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := auth.CurrentUser(c)
		if u == nil {
			httpx.WriteError(c, httpx.Unauthorized(msgAuthRequired))
			return
		}
		units, err := scopes.ResidentUnitIDs(c.Request.Context(), u.Phone)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		items, total, err := svc.repo.ListInvoicesForUnits(
			c.Request.Context(), units,
			atoiDefault(c.Query("page"), 1), atoiDefault(c.Query("page_size"), 20))
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"items": items, "total": total,
			"page": atoiDefault(c.Query("page"), 1), "page_size": atoiDefault(c.Query("page_size"), 20),
		})
	}
}

func myInvoice(svc *PeriodService, scopes *auth.ScopeResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := auth.CurrentUser(c)
		if u == nil {
			httpx.WriteError(c, httpx.Unauthorized(msgAuthRequired))
			return
		}
		id, ok := parseID(c, "id")
		if !ok {
			return
		}
		inv, err := svc.repo.GetInvoice(c.Request.Context(), id)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		units, err := scopes.ResidentUnitIDs(c.Request.Context(), u.Phone)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		for _, uid := range units {
			if uid == inv.UnitID {
				c.JSON(http.StatusOK, inv)
				return
			}
		}
		httpx.WriteError(c, httpx.Forbidden("شما به این صورتحساب دسترسی ندارید"))
	}
}

func atoiDefault(s string, d int) int {
	if s == "" {
		return d
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return d
		}
		n = n*10 + int(r-'0')
	}
	if n == 0 {
		return d
	}
	return n
}
