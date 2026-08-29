package payment

// T054 — US5 integration suite: drives the real Gin handlers against a
// throwaway PostgreSQL schema with the real migrations applied (same harness
// pattern as billing/building). Covers quickstart Scenario 4 and contracts/
// api.md "Payments & Balances":
//
//   - manual partial payment → invoice `partial`, balance reconciles (§9)
//   - paying the remainder → invoice `paid`, balance 0
//   - overpayment → surplus becomes unit credit applied to the next invoice
//   - gateway flow with the mock gateway: start → callback verify success /
//     failure / amount-mismatch (verify amount always taken from our DB)
//   - resident isolation: a resident only sees their own payments; manager
//     routes reject residents (403)

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/audit"
	"hamsa/internal/auth"
	"hamsa/internal/billing"
	"hamsa/internal/notification"
	"hamsa/internal/payment/gateway"
	"hamsa/internal/platform/db"
	"hamsa/internal/platform/httpx"
)

type payEnv struct {
	engine  *gin.Engine
	gormDB  *gorm.DB
	tokens  *auth.TokenService
	repo    *Repository
	bal     *BalanceService
	gateway *gateway.Mock
}

func (e *payEnv) exec(query string, args ...any) error {
	return e.gormDB.WithContext(context.Background()).Exec(query, args...).Error
}

const payBaseDSN = "host=localhost port=5432 user=hamsa password=hamsa dbname=hamsa sslmode=disable"

func newPaymentEnv(t *testing.T) *payEnv {
	t.Helper()

	dsn := os.Getenv("HAMSA_TEST_DSN")
	if dsn == "" {
		if !pgReachablePay(t) {
			t.Skip("skipping integration test — no Docker engine and no reachable PostgreSQL (set HAMSA_TEST_DSN)")
		}
		dsn = payBaseDSN
	}

	schema := "hamsa_test_" + strings.ReplaceAll(strings.ToLower(uuid.NewString()[:8]), "-", "")
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("skipping integration test — cannot reach PostgreSQL: %v", err)
	}
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		admin.Close()
		t.Skipf("skipping integration test — cannot create test schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		admin.Close()
	})

	dsn = dsn + " search_path=" + schema
	if err := db.Migrate(dsn, "../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	gormDB, err := db.Connect(dsn, log)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	tokens := auth.NewTokenService([]byte("test-secret-32-bytes-long!!!!!!"),
		15*time.Minute, 30*24*time.Hour, &auth.GormRefreshStore{DB: gormDB}, auth.RealClock{})
	repo := NewRepository(gormDB)
	notifSvc := notification.NewService(gormDB, notification.NoopNotifier{})
	aud := audit.New(gormDB, log)
	bal := NewBalanceService(repo)
	gw := gateway.NewMock()

	gin.SetMode(gin.TestMode)
	router := httpx.NewRouter(log, "dev")
	authMW := auth.Authenticate(tokens, auth.NewRepository(gormDB))
	svc := NewPaymentService(repo, gw, bal, notifSvc, log, "http://test/callback")
	authed := router.Group("/api/v1", authMW)
	public := router.Group("/api/v1")
	Register(authed, public, svc, bal, aud, auth.NewScopeResolver(gormDB))

	// Billing routes so the tests run against real issued invoices, with the
	// balance service wired for issue/cancel/adjust recomputes.
	bRepo := billing.NewRepository(gormDB)
	periodSvc := billing.NewPeriodService(bRepo, notifSvc, aud)
	periodSvc.SetBalanceRecomputer(bal)
	billing.Register(authed,
		billing.NewCalcService(bRepo, billingBalanceProvider{bal}),
		periodSvc,
		aud,
		auth.NewScopeResolver(gormDB))

	return &payEnv{engine: router, gormDB: gormDB, tokens: tokens, repo: repo, bal: bal, gateway: gw}
}

// billingBalanceProvider adapts the balance service to billing's
// BalanceProvider (same adapter main.go installs).
type billingBalanceProvider struct{ bal *BalanceService }

func (p billingBalanceProvider) Snapshot(ctx context.Context, unitID uuid.UUID) (billing.Money, billing.Money, error) {
	pd, cr, err := p.bal.Snapshot(ctx, unitID)
	return billing.Money(pd), billing.Money(cr), err
}

func pgReachablePay(t *testing.T) bool {
	t.Helper()
	dbh, err := sql.Open("pgx", payBaseDSN)
	if err != nil {
		return false
	}
	defer dbh.Close()
	dbh.SetConnMaxLifetime(time.Second)
	return dbh.Ping() == nil
}

func (e *payEnv) seedUser(t *testing.T, role, phone string) (uuid.UUID, string) {
	t.Helper()
	id := uuid.New()
	if err := e.exec(
		`INSERT INTO users (id, phone, role, name) VALUES (?, ?, ?, ?)`,
		id, phone, role, "تست "+role,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	token, _, err := e.tokens.IssueAccessToken(id)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return id, token
}

func (e *payEnv) get(t *testing.T, path, token string) (int, map[string]any) {
	t.Helper()
	return e.do(t, http.MethodGet, path, token, nil)
}

func (e *payEnv) post(t *testing.T, path, token string, body any) (int, map[string]any) {
	t.Helper()
	return e.do(t, http.MethodPost, path, token, body)
}

func (e *payEnv) do(t *testing.T, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rd = strings.NewReader(string(b))
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	var out map[string]any
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &out)
	}
	return w.Code, out
}

// seedBuildingUnit creates a building + one unit owned by the manager.
func (e *payEnv) seedBuildingUnit(t *testing.T, mgrID uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()
	bID, uID := uuid.New(), uuid.New()
	if err := e.exec(`INSERT INTO buildings (id, name, created_at, updated_at) VALUES (?, 'تست', now(), now())`, bID); err != nil {
		t.Fatalf("seed building: %v", err)
	}
	if err := e.exec(`INSERT INTO user_buildings (user_id, building_id) VALUES (?, ?)`, mgrID, bID); err != nil {
		t.Fatalf("seed grant: %v", err)
	}
	if err := e.exec(`INSERT INTO units (id, building_id, number, area_m2, status) VALUES (?, ?, '1', 100, 'active')`, uID, bID); err != nil {
		t.Fatalf("seed unit: %v", err)
	}
	return bID, uID
}

// issueInvoice drives calculate+issue through the billing handlers. The
// building has a single unit, so an equal cost item gives it the full amount.
func (e *payEnv) issueInvoice(t *testing.T, mgrToken string, bID, uID uuid.UUID, finalAmount int64) uuid.UUID {
	t.Helper()
	code, body := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/periods", bID), mgrToken, map[string]any{
		"title": "دوره", "start_date": "2026-03-21", "end_date": "2026-09-22", "due_date": "2026-09-22",
	})
	if code != http.StatusCreated {
		t.Fatalf("create period: %d %v", code, body)
	}
	periodID := body["id"].(string)
	code, body = e.post(t, fmt.Sprintf("/api/v1/periods/%s/cost-items", periodID), mgrToken, map[string]any{
		"title": "شارژ", "method": "equal", "total_amount": fmt.Sprintf("%d", finalAmount), "include_vacant": true,
	})
	if code != http.StatusCreated {
		t.Fatalf("create cost item: %d %v", code, body)
	}
	code, _ = e.post(t, fmt.Sprintf("/api/v1/periods/%s/calculate", periodID), mgrToken, nil)
	if code != http.StatusOK {
		t.Fatalf("calculate: %d", code)
	}
	code, body = e.post(t, fmt.Sprintf("/api/v1/periods/%s/issue", periodID), mgrToken, nil)
	if code != http.StatusOK {
		t.Fatalf("issue: %d %v", code, body)
	}
	items := body["items"].([]any)
	for _, it := range items {
		inv := it.(map[string]any)
		if inv["unit_id"].(string) == uID.String() {
			return uuid.MustParse(inv["id"].(string))
		}
	}
	t.Fatalf("no invoice for unit %s", uID)
	return uuid.Nil
}

func (e *payEnv) balance(t *testing.T, token, unitID string) map[string]any {
	t.Helper()
	code, body := e.get(t, "/api/v1/units/"+unitID+"/balance", token)
	if code != http.StatusOK {
		t.Fatalf("balance: %d %v", code, body)
	}
	return body
}

// num reads a JSON number or numeric string from a response map.
func num(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case string:
		var n int64
		fmt.Sscanf(v, "%d", &n)
		return n
	default:
		return 0
	}
}

// --- manual payment lifecycle ---------------------------------------------------

func TestManualPartialThenFullPayment(t *testing.T) {
	e := newPaymentEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000001")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	invID := e.issueInvoice(t, mgrTok, bID, unitID, 2_600_000)

	// §9 baseline: balance = current charge 2,600,000.
	b := e.balance(t, mgrTok, unitID.String())
	if got := num(b, "balance"); got != 2_600_000 {
		t.Fatalf("initial balance = %d, want 2600000", got)
	}

	// Partial 1,500,000 → invoice partial, outstanding 1,100,000.
	code, body := e.post(t, "/api/v1/invoices/"+invID.String()+"/payments", mgrTok, map[string]any{
		"amount": 1_500_000, "paid_at": "2026-08-29",
	})
	if code != http.StatusCreated {
		t.Fatalf("manual payment: %d %v", code, body)
	}
	code, inv := e.get(t, "/api/v1/invoices/"+invID.String(), mgrTok)
	if code != 200 || inv["status"] != "partial" || num(inv, "paid_amount") != 1_500_000 {
		t.Fatalf("after partial: %d %v", code, inv)
	}
	b = e.balance(t, mgrTok, unitID.String())
	if got := num(b, "balance"); got != 1_100_000 {
		t.Fatalf("balance after partial = %d, want 1100000", got)
	}
	// §9 reconciliation: prior + current + late − paid − credit = balance.
	if num(b, "prior_debt") != 0 || num(b, "current_invoice_amount") != 2_600_000 {
		t.Fatalf("components wrong: %v", b)
	}

	// Remainder 1,100,000 → invoice paid, balance 0.
	code, _ = e.post(t, "/api/v1/invoices/"+invID.String()+"/payments", mgrTok, map[string]any{
		"amount": 1_100_000, "paid_at": "2026-08-29",
	})
	if code != http.StatusCreated {
		t.Fatalf("second payment: %d", code)
	}
	code, inv = e.get(t, "/api/v1/invoices/"+invID.String(), mgrTok)
	if code != 200 || inv["status"] != "paid" {
		t.Fatalf("after settle: %d %v", code, inv)
	}
	b = e.balance(t, mgrTok, unitID.String())
	if got := num(b, "balance"); got != 0 {
		t.Fatalf("balance after settle = %d, want 0", got)
	}
}

func TestOverpaymentBecomesCredit(t *testing.T) {
	e := newPaymentEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000002")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	invID := e.issueInvoice(t, mgrTok, bID, unitID, 2_600_000)

	// Overpay by 400,000 → invoice paid, surplus stored as credit asset.
	code, body := e.post(t, "/api/v1/invoices/"+invID.String()+"/payments", mgrTok, map[string]any{
		"amount": 3_000_000, "paid_at": "2026-08-29",
	})
	if code != http.StatusCreated {
		t.Fatalf("overpay: %d %v", code, body)
	}
	code, inv := e.get(t, "/api/v1/invoices/"+invID.String(), mgrTok)
	if code != 200 || inv["status"] != "paid" || num(inv, "paid_amount") != 2_600_000 {
		t.Fatalf("after overpay invoice: %d %v", code, inv)
	}
	b := e.balance(t, mgrTok, unitID.String())
	if got := num(b, "balance"); got != 0 {
		t.Fatalf("balance after overpay = %d, want 0", got)
	}
	if got := num(b, "credit_asset"); got != 400_000 {
		t.Fatalf("credit_asset = %d, want 400000", got)
	}

	// Next period's invoice must apply the credit (credit_amount 400,000).
	inv2 := e.issueInvoice(t, mgrTok, bID, unitID, 1_000_000)
	code, inv = e.get(t, "/api/v1/invoices/"+inv2.String(), mgrTok)
	if code != 200 || num(inv, "credit_amount") != 400_000 || num(inv, "final_amount") != 600_000 {
		t.Fatalf("next invoice should embed credit: %d %v", code, inv)
	}
	b = e.balance(t, mgrTok, unitID.String())
	// Σ(final − prior) − paid over open invoices = 600000 (asset consumed).
	if got := num(b, "balance"); got != 600_000 {
		t.Fatalf("balance after credit applied = %d, want 600000", got)
	}
}

func TestManualPaymentValidation(t *testing.T) {
	e := newPaymentEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000003")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	invID := e.issueInvoice(t, mgrTok, bID, unitID, 100_000)

	// Zero/negative amount rejected.
	for _, amt := range []int{0, -5} {
		if code, _ := e.post(t, "/api/v1/invoices/"+invID.String()+"/payments", mgrTok,
			map[string]any{"amount": amt, "paid_at": "2026-08-29"}); code != http.StatusBadRequest {
			t.Fatalf("amount %d: code %d, want 400", amt, code)
		}
	}
	// Resident cannot record manual payments.
	e.grantResidency(t, bID, unitID, "09120000004")
	_, resTok := e.seedUser(t, "resident", "09120000004")
	if code, _ := e.post(t, "/api/v1/invoices/"+invID.String()+"/payments", resTok,
		map[string]any{"amount": 1000, "paid_at": "2026-08-29"}); code != http.StatusForbidden {
		t.Fatalf("resident manual payment: %d, want 403", code)
	}
}

// --- gateway flow ---------------------------------------------------------------

// startPay starts a gateway payment as the resident and returns (code, body).
func startPay(t *testing.T, e *payEnv, resTok, invID string) (int, map[string]any) {
	t.Helper()
	return e.post(t, "/api/v1/invoices/"+invID+"/pay", resTok, map[string]any{})
}

// verifyCallback hits the public callback with the payment's authority.
func (e *payEnv) verifyCallback(t *testing.T, payID string) (int, map[string]any) {
	t.Helper()
	var authority string
	if err := e.gormDB.WithContext(context.Background()).
		Raw(`SELECT authority FROM payments WHERE id = ?`, uuid.MustParse(payID)).
		Scan(&authority).Error; err != nil || authority == "" {
		t.Fatalf("lookup authority: %v %q", err, authority)
	}
	return e.get(t, "/api/v1/payments/callback?Authority="+authority+"&Status=OK", "")
}

func TestGatewayPaymentSuccess(t *testing.T) {
	e := newPaymentEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000005")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	_ = mgrID
	e.grantResidency(t, bID, unitID, "09120000006")
	_, resTok := e.seedUser(t, "resident", "09120000006")
	invID := e.issueInvoice(t, mgrTok, bID, unitID, 2_600_000)

	// Resident starts a gateway payment for the full outstanding amount.
	code, body := startPay(t, e, resTok, invID.String())
	if code != http.StatusCreated {
		t.Fatalf("start pay: %d %v", code, body)
	}
	payURL, _ := body["payment_url"].(string)
	payID, _ := body["payment_id"].(string)
	if payURL == "" || payID == "" {
		t.Fatalf("missing payment_url/payment_id: %v", body)
	}

	// Callback verify (amount comes from our DB, not the callback).
	code, body = e.verifyCallback(t, payID)
	if code != http.StatusOK {
		t.Fatalf("callback: %d %v", code, body)
	}
	if body["status"] != "verified" {
		t.Fatalf("payment not verified: %v", body)
	}
	code, inv := e.get(t, "/api/v1/invoices/"+invID.String(), mgrTok)
	if code != 200 || inv["status"] != "paid" {
		t.Fatalf("invoice after gateway pay: %d %v", code, inv)
	}
	b := e.balance(t, mgrTok, unitID.String())
	if got := num(b, "balance"); got != 0 {
		t.Fatalf("balance after gateway pay = %d, want 0", got)
	}

	// Resident sees the payment in /me/payments.
	code, body = e.get(t, "/api/v1/me/payments", resTok)
	if code != 200 {
		t.Fatalf("me/payments: %d", code)
	}
	items := body["payments"].([]any)
	if len(items) != 1 {
		t.Fatalf("me/payments items = %d, want 1", len(items))
	}
}

func TestGatewayPaymentFailureAndMismatch(t *testing.T) {
	e := newPaymentEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000007")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	e.grantResidency(t, bID, unitID, "09120000008")
	_, resTok := e.seedUser(t, "resident", "09120000008")
	invID := e.issueInvoice(t, mgrTok, bID, unitID, 1_000_000)

	// Gateway failure at verify → payment failed, invoice untouched.
	e.gateway.FailNext = true
	code, body := startPay(t, e, resTok, invID.String())
	if code != http.StatusCreated {
		t.Fatalf("start pay: %d %v", code, body)
	}
	payID := body["payment_id"].(string)
	code, body = e.verifyCallback(t, payID)
	if code != http.StatusOK || body["status"] != "failed" {
		t.Fatalf("failed callback: %d %v, want 200/failed", code, body)
	}
	code, inv := e.get(t, "/api/v1/invoices/"+invID.String(), mgrTok)
	if code != 200 || inv["status"] != "unpaid" || num(inv, "paid_amount") != 0 {
		t.Fatalf("invoice must be untouched after failure: %d %v", code, inv)
	}

	// Amount-mismatch: the gateway verify is called with the DB amount; the
	// mock mismatch hook simulates the gateway rejecting a tampered amount.
	e.gateway.MismatchNext = true
	code, body = startPay(t, e, resTok, invID.String())
	if code != http.StatusCreated {
		t.Fatalf("start pay 2: %d %v", code, body)
	}
	payID = body["payment_id"].(string)
	code, body = e.verifyCallback(t, payID)
	if code != http.StatusOK || body["status"] != "failed" {
		t.Fatalf("mismatch callback: %d %v, want 200/failed", code, body)
	}

	// A successful retry still works.
	code, body = startPay(t, e, resTok, invID.String())
	if code != http.StatusCreated {
		t.Fatalf("start pay 3: %d %v", code, body)
	}
	payID = body["payment_id"].(string)
	if code, body = e.verifyCallback(t, payID); code != http.StatusOK || body["status"] != "verified" {
		t.Fatalf("retry callback: %d %v", code, body)
	}
	code, inv = e.get(t, "/api/v1/invoices/"+invID.String(), mgrTok)
	if code != 200 || inv["status"] != "paid" {
		t.Fatalf("invoice after retry: %d %v", code, inv)
	}
}

func TestPaymentIsolation(t *testing.T) {
	e := newPaymentEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000009")
	bID, unitA := e.seedBuildingUnit(t, mgrID)
	invA := e.issueInvoice(t, mgrTok, bID, unitA, 100_000)
	e.grantResidency(t, bID, unitA, "09120000010")
	_, tokA := e.seedUser(t, "resident", "09120000010")

	// Resident A pays own invoice via gateway.
	code, body := startPay(t, e, tokA, invA.String())
	if code != http.StatusCreated {
		t.Fatalf("start pay: %d %v", code, body)
	}
	payID := body["payment_id"].(string)
	if code, _ = e.verifyCallback(t, payID); code != http.StatusOK {
		t.Fatalf("callback: %d", code)
	}

	// Another resident must not see A's balance; residents cannot open the
	// manager ledger.
	_, tokB := e.seedUser(t, "resident", "09120000011")
	if code, _ := e.get(t, "/api/v1/units/"+unitA.String()+"/balance", tokB); code != http.StatusForbidden {
		t.Fatalf("unit balance B: %d, want 403", code)
	}
	if code, _ := e.get(t, "/api/v1/buildings/"+bID.String()+"/payments", tokA); code != http.StatusForbidden {
		t.Fatalf("ledger as resident: %d, want 403", code)
	}
	// B's own payment history is empty but readable.
	code, body = e.get(t, "/api/v1/me/payments", tokB)
	if code != 200 {
		t.Fatalf("me/payments B: %d", code)
	}
	if total := num(body, "total"); total != 0 {
		t.Fatalf("B payment total = %d, want 0", total)
	}
}

// --- helpers ---------------------------------------------------------------------

func (e *payEnv) grantResidency(t *testing.T, bID, unitID uuid.UUID, phone string) {
	t.Helper()
	pid := uuid.New()
	if err := e.exec(`INSERT INTO persons (id, building_id, full_name, phone) VALUES (?, ?, 'ساکن', ?)`,
		pid, bID, phone); err != nil {
		t.Fatalf("seed person: %v", err)
	}
	if err := e.exec(`INSERT INTO occupancies (id, person_id, unit_id, relationship, start_date) VALUES (?, ?, ?, 'tenant', '2026-01-01')`,
		uuid.New(), pid, unitID); err != nil {
		t.Fatalf("seed occupancy: %v", err)
	}
}
