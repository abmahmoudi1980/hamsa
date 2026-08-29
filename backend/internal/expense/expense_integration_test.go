package expense

// T062 — US6 integration suite: drives the real Gin handlers against a
// throwaway PostgreSQL schema with the real migrations applied (same harness
// pattern as billing/building/payment). Covers contracts/api.md
// "Expenses & Financial Report" and quickstart Scenario 5:
//
//   - expense CRUD with receipt file binding (T010 files), category enum,
//     approval workflow (pending → approved/rejected via PUT)
//   - manager-scope isolation: a manager without the building grant gets 403;
//     residents are rejected on manager routes
//   - soft delete: the row survives, lists and the report exclude it
//   - financial report aggregation accuracy vs seeded payments/expenses/
//     balances (FR-027): monthly income/expense/net + building-wide totals
//   - audit trail for expense.record / expense.update (FR-038)

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
	"hamsa/internal/platform/db"
	"hamsa/internal/platform/httpx"
)

type expEnv struct {
	engine *gin.Engine
	gormDB *gorm.DB
	tokens *auth.TokenService
}

func (e *expEnv) exec(query string, args ...any) error {
	return e.gormDB.WithContext(context.Background()).Exec(query, args...).Error
}

const expBaseDSN = "host=localhost port=5432 user=hamsa password=hamsa dbname=hamsa sslmode=disable"

func newExpenseEnv(t *testing.T) *expEnv {
	t.Helper()

	dsn := os.Getenv("HAMSA_TEST_DSN")
	if dsn == "" {
		if !pgReachableExp(t) {
			t.Skip("skipping integration test — no Docker engine and no reachable PostgreSQL (set HAMSA_TEST_DSN)")
		}
		dsn = expBaseDSN
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

	gin.SetMode(gin.TestMode)
	router := httpx.NewRouter(log, "dev")
	authMW := auth.Authenticate(tokens, auth.NewRepository(gormDB))
	authed := router.Group("/api/v1", authMW)
	Register(authed, NewService(NewRepository(gormDB)), audit.New(gormDB, log))

	return &expEnv{engine: router, gormDB: gormDB, tokens: tokens}
}

func pgReachableExp(t *testing.T) bool {
	t.Helper()
	dbh, err := sql.Open("pgx", expBaseDSN)
	if err != nil {
		return false
	}
	defer dbh.Close()
	dbh.SetConnMaxLifetime(time.Second)
	return dbh.Ping() == nil
}

func (e *expEnv) seedUser(t *testing.T, role, phone string) (uuid.UUID, string) {
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

// seedBuilding creates a building granted to the manager (no units needed —
// expenses attach to the building only).
func (e *expEnv) seedBuilding(t *testing.T, mgrID uuid.UUID) uuid.UUID {
	t.Helper()
	bID := uuid.New()
	if err := e.exec(`INSERT INTO buildings (id, name) VALUES (?, 'برج هزینه')`, bID); err != nil {
		t.Fatalf("seed building: %v", err)
	}
	if err := e.exec(`INSERT INTO user_buildings (user_id, building_id) VALUES (?, ?)`, mgrID, bID); err != nil {
		t.Fatalf("seed grant: %v", err)
	}
	return bID
}

// seedUnit inserts a unit plus a unit_balances row (BR-01) with the given
// outstanding balance — the report's total-debt source of truth.
func (e *expEnv) seedUnitWithBalance(t *testing.T, bID uuid.UUID, number string, balance int64) uuid.UUID {
	t.Helper()
	uID := uuid.New()
	if err := e.exec(
		`INSERT INTO units (id, building_id, number, area_m2, status) VALUES (?, ?, ?, 100, 'active')`,
		uID, bID, number,
	); err != nil {
		t.Fatalf("seed unit: %v", err)
	}
	if err := e.exec(
		`INSERT INTO unit_balances (unit_id, balance) VALUES (?, ?)`,
		uID, balance,
	); err != nil {
		t.Fatalf("seed balance: %v", err)
	}
	return uID
}

// seedPayment inserts a payments row with an explicit status — the report
// counts only verified rows.
func (e *expEnv) seedPayment(t *testing.T, bID, uID uuid.UUID, amount int64, paidAt, status string) {
	t.Helper()
	if err := e.exec(
		`INSERT INTO payments (id, building_id, unit_id, method, amount, paid_at, status)
		 VALUES (?, ?, ?, 'manual', ?, ?, ?)`,
		uuid.New(), bID, uID, amount, paidAt, status,
	); err != nil {
		t.Fatalf("seed payment: %v", err)
	}
}

// seedReceipt inserts a files row (T010 registry) and returns its id.
func (e *expEnv) seedReceipt(t *testing.T, mgrID uuid.UUID) uuid.UUID {
	t.Helper()
	fID := uuid.New()
	if err := e.exec(
		`INSERT INTO files (id, path, content_type, size_bytes, uploaded_by) VALUES (?, ?, 'image/jpeg', 1234, ?)`,
		fID, "uploads/"+fID.String()+".jpg", mgrID,
	); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	return fID
}

func (e *expEnv) do(t *testing.T, method, path, token string, body any) (int, map[string]any) {
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

func (e *expEnv) get(t *testing.T, path, token string) (int, map[string]any) {
	t.Helper()
	return e.do(t, http.MethodGet, path, token, nil)
}

func (e *expEnv) post(t *testing.T, path, token string, body any) (int, map[string]any) {
	t.Helper()
	return e.do(t, http.MethodPost, path, token, body)
}

func (e *expEnv) put(t *testing.T, path, token string, body any) (int, map[string]any) {
	t.Helper()
	return e.do(t, http.MethodPut, path, token, body)
}

func (e *expEnv) del(t *testing.T, path, token string) (int, map[string]any) {
	t.Helper()
	return e.do(t, http.MethodDelete, path, token, nil)
}

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

func createExpenseRow(t *testing.T, e *expEnv, token, bID string, body map[string]any) string {
	t.Helper()
	if body == nil {
		body = map[string]any{}
	}
	code, resp := e.post(t, "/api/v1/buildings/"+bID+"/expenses", token, body)
	if code != http.StatusCreated {
		t.Fatalf("create expense: %d %v", code, resp)
	}
	return resp["id"].(string)
}

// --- CRUD, receipt binding, scope, soft delete --------------------------------

func TestExpenseCRUDScopeAndSoftDelete(t *testing.T) {
	e := newExpenseEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000001")
	bID := e.seedBuilding(t, mgrID)
	receipt := e.seedReceipt(t, mgrID)

	// Create with receipt → 201, fields echoed, default approval pending.
	code, resp := e.post(t, "/api/v1/buildings/"+bID.String()+"/expenses", mgrTok, map[string]any{
		"title": "برق لابی", "category": "electricity", "amount": 2_000_000,
		"expense_date": "2026-06-05", "description": "قبض مرداد",
		"receipt_file_id": receipt.String(),
	})
	if code != http.StatusCreated {
		t.Fatalf("create: %d %v", code, resp)
	}
	if resp["approval_status"] != "pending" {
		t.Fatalf("default approval = %v, want pending", resp["approval_status"])
	}
	expID := resp["id"].(string)

	// Unknown receipt file → 400.
	code, resp = e.post(t, "/api/v1/buildings/"+bID.String()+"/expenses", mgrTok, map[string]any{
		"title": "بدون فایل", "category": "other", "amount": 1,
		"expense_date": "2026-06-06", "receipt_file_id": uuid.NewString(),
	})
	if code != http.StatusBadRequest {
		t.Fatalf("unknown receipt: %d %v", code, resp)
	}

	// Invalid amount (≤ 0) → 400.
	code, resp = e.post(t, "/api/v1/buildings/"+bID.String()+"/expenses", mgrTok, map[string]any{
		"title": "منفی", "category": "other", "amount": -5, "expense_date": "2026-06-06",
	})
	if code != http.StatusBadRequest {
		t.Fatalf("negative amount: %d %v", code, resp)
	}

	// Invalid category → 400.
	code, resp = e.post(t, "/api/v1/buildings/"+bID.String()+"/expenses", mgrTok, map[string]any{
		"title": "بد", "category": "luxury", "amount": 10, "expense_date": "2026-06-06",
	})
	if code != http.StatusBadRequest {
		t.Fatalf("bad category: %d %v", code, resp)
	}

	// A manager without the building grant gets 403 (create, update, report).
	_, otherTok := e.seedUser(t, "manager", "09120000002")
	if code, _ := e.post(t, "/api/v1/buildings/"+bID.String()+"/expenses", otherTok, map[string]any{
		"title": "خارج از دسترس", "category": "other", "amount": 1, "expense_date": "2026-06-06",
	}); code != http.StatusForbidden {
		t.Fatalf("foreign manager create: %d", code)
	}
	if code, _ := e.put(t, "/api/v1/expenses/"+expID, otherTok, map[string]any{"amount": 9}); code != http.StatusForbidden {
		t.Fatalf("foreign manager update: %d", code)
	}
	if code, _ := e.get(t, "/api/v1/buildings/"+bID.String()+"/financial-report?month=2026-06", otherTok); code != http.StatusForbidden {
		t.Fatalf("foreign manager report: %d", code)
	}

	// Residents are rejected on manager-only expense routes.
	_, resTok := e.seedUser(t, "resident", "09120000003")
	if code, _ := e.get(t, "/api/v1/buildings/"+bID.String()+"/expenses", resTok); code != http.StatusForbidden {
		t.Fatalf("resident list: %d", code)
	}

	// Update: amount + approval workflow pending → approved → rejected.
	code, resp = e.put(t, "/api/v1/expenses/"+expID, mgrTok, map[string]any{
		"amount": 2_500_000, "approval_status": "approved",
	})
	if code != http.StatusOK || resp["approval_status"] != "approved" || num(resp, "amount") != 2_500_000 {
		t.Fatalf("approve: %d %v", code, resp)
	}
	code, resp = e.put(t, "/api/v1/expenses/"+expID, mgrTok, map[string]any{
		"approval_status": "rejected",
	})
	if code != http.StatusOK || resp["approval_status"] != "rejected" {
		t.Fatalf("reject: %d %v", code, resp)
	}

	// List with filters: category + approval + date window.
	code, resp = e.post(t, "/api/v1/buildings/"+bID.String()+"/expenses", mgrTok, map[string]any{
		"title": "نگهبانی", "category": "security", "amount": 8_000_000, "expense_date": "2026-07-01",
	})
	if code != http.StatusCreated {
		t.Fatalf("second expense: %d %v", code, resp)
	}
	code, resp = e.get(t, "/api/v1/buildings/"+bID.String()+"/expenses?category=electricity", mgrTok)
	if code != 200 || len(resp["items"].([]any)) != 1 {
		t.Fatalf("filter category: %d %v", code, resp)
	}
	code, resp = e.get(t, "/api/v1/buildings/"+bID.String()+"/expenses?approval=rejected", mgrTok)
	if code != 200 || len(resp["items"].([]any)) != 1 {
		t.Fatalf("filter approval: %d %v", code, resp)
	}
	code, resp = e.get(t, "/api/v1/buildings/"+bID.String()+"/expenses?from=2026-07-01&to=2026-07-31", mgrTok)
	if code != 200 || len(resp["items"].([]any)) != 1 {
		t.Fatalf("filter dates: %d %v", code, resp)
	}

	// Soft delete: row survives, disappears from list and single GET → 404.
	if code, _ := e.del(t, "/api/v1/expenses/"+expID, mgrTok); code != http.StatusOK {
		t.Fatalf("delete: %d", code)
	}
	code, resp = e.get(t, "/api/v1/buildings/"+bID.String()+"/expenses", mgrTok)
	if code != 200 || len(resp["items"].([]any)) != 1 {
		t.Fatalf("list after delete: %d %v", code, resp)
	}
	var n int64
	if err := e.gormDB.Raw(`SELECT count(*) FROM expenses WHERE id = ?`, expID).Scan(&n).Error; err != nil || n != 1 {
		t.Fatalf("soft-deleted row must survive: n=%d err=%v", n, err)
	}
	if code, _ := e.get(t, "/api/v1/expenses/"+expID, mgrTok); code != http.StatusNotFound {
		t.Fatalf("get soft-deleted: %d, want 404", code)
	}
}

// --- audit trail (FR-038: expense record) -------------------------------------

func TestExpenseAuditTrail(t *testing.T) {
	e := newExpenseEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000011")
	bID := e.seedBuilding(t, mgrID)
	expID := createExpenseRow(t, e, mgrTok, bID.String(), map[string]any{
		"title": "آسانسور", "category": "elevator", "amount": 500_000, "expense_date": "2026-06-10",
	})

	if code, _ := e.put(t, "/api/v1/expenses/"+expID, mgrTok, map[string]any{"approval_status": "approved"}); code != http.StatusOK {
		t.Fatalf("update: %d", code)
	}

	var actions []string
	if err := e.gormDB.Raw(
		`SELECT action FROM audit_logs WHERE user_id = ? AND object_id = ? ORDER BY created_at`,
		mgrID, expID,
	).Scan(&actions).Error; err != nil {
		t.Fatalf("audit query: %v", err)
	}
	if len(actions) != 2 || actions[0] != "expense.record" || actions[1] != "expense.update" {
		t.Fatalf("audit actions = %v, want [expense.record expense.update]", actions)
	}
}

// --- financial report aggregation (FR-027, quickstart Scenario 5) --------------

func TestFinancialReportAggregation(t *testing.T) {
	e := newExpenseEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000021")
	bID := e.seedBuilding(t, mgrID)
	u1 := e.seedUnitWithBalance(t, bID, "1", 1_100_000)
	u2 := e.seedUnitWithBalance(t, bID, "2", 500_000)

	// Payments: June verified 1,500,000 + July verified 300,000; a reversed
	// June row of 999,000 must be excluded from every total.
	e.seedPayment(t, bID, u1, 1_500_000, "2026-06-10", "verified")
	e.seedPayment(t, bID, u2, 300_000, "2026-07-01", "verified")
	e.seedPayment(t, bID, u2, 999_000, "2026-06-15", "reversed")

	// Expenses: two June rows kept (2,000,000 + 8,000,000), one June row
	// soft-deleted (100,000, excluded), one July row (250,000).
	e.post(t, "/api/v1/buildings/"+bID.String()+"/expenses", mgrTok, map[string]any{
		"title": "برق لابی", "category": "electricity", "amount": 2_000_000,
		"expense_date": "2026-06-05",
	})
	secID := createExpenseRow(t, e, mgrTok, bID.String(), map[string]any{
		"title": "نگهبانی", "category": "security", "amount": 8_000_000,
		"expense_date": "2026-06-20",
	})
	delID := createExpenseRow(t, e, mgrTok, bID.String(), map[string]any{
		"title": "حذف‌شده", "category": "other", "amount": 100_000,
		"expense_date": "2026-06-25",
	})
	if code, _ := e.del(t, "/api/v1/expenses/"+delID, mgrTok); code != http.StatusOK {
		t.Fatalf("delete expense: %d", code)
	}
	createExpenseRow(t, e, mgrTok, bID.String(), map[string]any{
		"title": "گاز", "category": "gas", "amount": 250_000, "expense_date": "2026-07-05",
	})
	_ = secID

	// June report: income 1,500,000, expense 10,000,000, net −8,500,000;
	// building-wide: debt 1,600,000, payments 1,800,000, expenses 10,250,000.
	code, rep := e.get(t, "/api/v1/buildings/"+bID.String()+"/financial-report?month=2026-06", mgrTok)
	if code != http.StatusOK {
		t.Fatalf("report: %d %v", code, rep)
	}
	want := map[string]int64{
		"monthly_income":  1_500_000,
		"monthly_expense": 10_000_000,
		"net":             -8_500_000,
		"total_debt":      1_600_000,
		"total_payments":  1_800_000,
		"total_expenses":  10_250_000,
	}
	for k, v := range want {
		if got := num(rep, k); got != v {
			t.Fatalf("report %s = %d, want %d (full: %v)", k, got, v, rep)
		}
	}

	// July report: income 300,000, expense 250,000.
	code, rep = e.get(t, "/api/v1/buildings/"+bID.String()+"/financial-report?month=2026-07", mgrTok)
	if code != http.StatusOK {
		t.Fatalf("july report: %d %v", code, rep)
	}
	if num(rep, "monthly_income") != 300_000 || num(rep, "monthly_expense") != 250_000 || num(rep, "net") != 50_000 {
		t.Fatalf("july report mismatch: %v", rep)
	}

	// Malformed month → 400.
	if code, _ = e.get(t, "/api/v1/buildings/"+bID.String()+"/financial-report?month=2026-13", mgrTok); code != http.StatusBadRequest {
		t.Fatalf("bad month: %d", code)
	}

	// A second building's data never leaks in.
	otherB := e.seedBuilding(t, mgrID)
	e.seedUnitWithBalance(t, otherB, "1", 7_000_000)
	code, rep = e.get(t, "/api/v1/buildings/"+bID.String()+"/financial-report?month=2026-06", mgrTok)
	if code != http.StatusOK || num(rep, "total_debt") != 1_600_000 {
		t.Fatalf("scope leak in report: %d %v", code, rep)
	}
}
