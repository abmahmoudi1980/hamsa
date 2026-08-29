package payment

// Handlers: US5 routes (contracts/api.md "Payments & Balances"). authed is
// the /api/v1 group with auth middleware; public is the same prefix without
// auth — the gateway cannot present a bearer token on its redirect.

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

// Register mounts US5 routes. The scopes resolver is kept in the signature
// for symmetric wiring with the other domains; object-level authz lives in
// the service (manager grant or active residency, FR-037).
func Register(authed, public *gin.RouterGroup, svc *PaymentService, balances *BalanceService,
	aud *audit.Service, scopes *auth.ScopeResolver) {
	_ = balances
	_ = scopes
	authed.POST("/invoices/:id/payments", aud.Middleware("payment.record", "payment"), recordManual(svc))
	authed.POST("/invoices/:id/pay", startGateway(svc))
	authed.GET("/buildings/:id/payments", buildingLedger(svc))
	authed.GET("/me/payments", myPayments(svc))
	authed.GET("/units/:id/balance", unitBalance(svc))
	public.GET("/payments/callback", gatewayCallback(svc))
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
// as the billing handlers).
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

// --- manual payment (manager) ------------------------------------------------------

func recordManual(svc *PaymentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		invoiceID, ok := parseID(c, "id")
		if !ok {
			return
		}
		var in ManualPaymentInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest("داده‌های پرداخت نامعتبر است"))
			return
		}
		p, err := svc.RecordManual(c.Request.Context(), u, invoiceID, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusCreated, p)
	}
}

// --- gateway start (resident or manager) ---------------------------------------------

func startGateway(svc *PaymentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		invoiceID, ok := parseID(c, "id")
		if !ok {
			return
		}
		var body struct {
			Amount *string `json:"amount"`
		}
		_ = c.ShouldBindJSON(&body) // body optional — defaults to outstanding
		var amount *Money
		if body.Amount != nil {
			v, err := strconv.ParseInt(*body.Amount, 10, 64)
			if err != nil {
				httpx.WriteError(c, httpx.BadRequest("مبلغ نامعتبر است"))
				return
			}
			amount = &v
		}
		p, payURL, err := svc.StartGateway(c.Request.Context(), u, invoiceID, amount)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"payment_id":  p.ID,
			"payment_url": payURL,
			"amount":      strconv.FormatInt(p.Amount, 10),
			"status":      p.Status,
		})
	}
}

// --- gateway callback (public) ----------------------------------------------------------

func gatewayCallback(svc *PaymentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Mock/dev flow identifies the payment directly; real gateways
		// (Zarinpal v4) return their Authority token, resolved against our
		// in-flight payment row — never trusting callback content.
		p, err := svc.HandleCallback(c.Request.Context(),
			c.Query("payment_id"), firstNonEmpty(c.Query("Authority"), c.Query("authority")))
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		invoiceID := ""
		if p.InvoiceID != nil {
			invoiceID = p.InvoiceID.String()
		}
		c.JSON(http.StatusOK, gin.H{
			"payment_id": p.ID,
			"status":     p.Status,
			"invoice_id": invoiceID,
			"deeplink":   "hamsa://payments/" + p.ID.String(),
		})
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// --- ledger ----------------------------------------------------------------------------

func buildingLedger(svc *PaymentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c, "id")
		if !ok {
			return
		}
		f := PaymentFilter{Page: atoiDefault(c.Query("page"), 1), Size: atoiDefault(c.Query("size"), 20)}
		if s := c.Query("unit_id"); s != "" {
			if uid, err := uuid.Parse(s); err == nil {
				f.UnitID = uid
			}
		}
		f.Method = c.Query("method")
		if v := c.Query("from"); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				f.From = &t
			}
		}
		if v := c.Query("to"); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				end := t.Add(24 * time.Hour)
				f.To = &end
			}
		}
		rows, total, err := svc.BuildingLedger(c.Request.Context(), u, buildingID, f)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"payments": rows, "total": total})
	}
}

func myPayments(svc *PaymentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		f := PaymentFilter{Page: atoiDefault(c.Query("page"), 1), Size: atoiDefault(c.Query("size"), 20)}
		f.Method = c.Query("method")
		rows, total, err := svc.MyPayments(c.Request.Context(), u, f)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"payments": rows, "total": total})
	}
}

// --- unit balance -------------------------------------------------------------------------

func unitBalance(svc *PaymentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := currentUser(c)
		if !ok {
			return
		}
		unitID, ok := parseID(c, "id")
		if !ok {
			return
		}
		b, err := svc.UnitBalanceFor(c.Request.Context(), u, unitID)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, b)
	}
}
