package dashboard

// T081 + T085 — US9 resident home + US10 manager dashboard integration suite.
//
// Drives the real Gin handlers against a throwaway PostgreSQL schema with the
// real migrations applied. Covers contracts/api.md "Resident Panel P0-08" and
// "Manager Dashboard P0-09", and the quickstart Scenarios 8.1 + 8.2:
//
//   - GET /me/home aggregates payable amount, latest invoice (id, amount, due
//     date, status), open request count + latest request status, latest 5
//     announcements, and unread announcement count, strictly scoped to the
//     resident's active units (FR-034, FR-037).
//   - GET /buildings/{id}/dashboard returns unit count, debtor units, total
//     debt, month income, month expense, open request count, pending expenses
//     and the alert list with the correct kinds/counts (FR-035).
//   - Resident isolation: a resident cannot read another resident's data via
//     /me/home (object-level authorization — no per-unit id in the path).
//   - Manager scope: a non-granted manager gets 403 on /buildings/{id}/dashboard;
//     residents get 403 on the manager route.

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

	"hamsa/internal/announcement"
	"hamsa/internal/audit"
	"hamsa/internal/auth"
	"hamsa/internal/maintenance"
	"hamsa/internal/notification"
	"hamsa/internal/platform/db"
	"hamsa/internal/platform/httpx"
)

type dashEnv struct {
	engine  *gin.Engine
	gormDB  *gorm.DB
	tokens  *auth.TokenService
	svc     *Service
	annRepo *announcement.Repository
}

func (e *dashEnv) exec(query string, args ...any) error {
	return e.gormDB.WithContext(context.Background()).Exec(query, args...).Error
}

const dashBaseDSN = "host=localhost port=5432 user=hamsa password=hamsa dbname=hamsa sslmode=disable"

func newDashEnv(t *testing.T) *dashEnv {
	t.Helper()

	dsn := os.Getenv("HAMSA_TEST_DSN")
	if dsn == "" {
		if !pgReachableDash(t) {
			t.Skip("skipping integration test — no Docker engine and no reachable PostgreSQL (set HAMSA_TEST_DSN)")
		}
		dsn = dashBaseDSN
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

	notifSvc := notification.NewService(gormDB, notification.NoopNotifier{})
	audSvc := audit.New(gormDB, log)
	annRepo := announcement.NewRepository(gormDB)
	maintRepo := maintenance.NewRepository(gormDB)
	annSvc := announcement.NewService(annRepo, notifSvc, audSvc)
	svc := NewService(gormDB, auth.NewScopeResolver(gormDB), annSvc, annRepo, maintRepo)

	gin.SetMode(gin.TestMode)
	router := httpx.NewRouter(log, "dev")
	authMW := auth.Authenticate(tokens, auth.NewRepository(gormDB))
	authed := router.Group("/api/v1", authMW)
	Register(authed, svc)

	return &dashEnv{engine: router, gormDB: gormDB, tokens: tokens, svc: svc, annRepo: annRepo}
}

func pgReachableDash(t *testing.T) bool {
	t.Helper()
	dbh, err := sql.Open("pgx", dashBaseDSN)
	if err != nil {
		return false
	}
	defer dbh.Close()
	dbh.SetConnMaxLifetime(time.Second)
	return dbh.Ping() == nil
}

func (e *dashEnv) seedUser(t *testing.T, role, phone string) (uuid.UUID, string) {
	t.Helper()
	id := uuid.New()
	if err := e.exec(`INSERT INTO users (id, phone, role, name, is_active) VALUES (?, ?, ?, ?, TRUE)`, id, phone, role, role+"-"+phone[len(phone)-4:]); err != nil {
		t.Fatalf("seedUser: %v", err)
	}
	tok, _, err := e.tokens.IssueAccessToken(id)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return id, tok
}

func (e *dashEnv) seedBuilding(t *testing.T, mgrID uuid.UUID) uuid.UUID {
	t.Helper()
	bID := uuid.New()
	if err := e.exec(`INSERT INTO buildings (id, name, address, block_count, floor_count, unit_count) VALUES (?, ?, ?, 2, 5, 10)`, bID, "برج هامسا", "تهران"); err != nil {
		t.Fatalf("seedBuilding: %v", err)
	}
	if err := e.exec(`INSERT INTO user_buildings (user_id, building_id) VALUES (?, ?)`, mgrID, bID); err != nil {
		t.Fatalf("grant building: %v", err)
	}
	return bID
}

func (e *dashEnv) seedUnit(t *testing.T, bID uuid.UUID, number string, status string) uuid.UUID {
	t.Helper()
	uID := uuid.New()
	if err := e.exec(`INSERT INTO units (id, building_id, number, floor, area_m2, status) VALUES (?, ?, ?, 1, 80, ?)`, uID, bID, number, status); err != nil {
		t.Fatalf("seedUnit %s: %v", number, err)
	}
	// Initialize a zero unit_balance row so SUM/COUNT queries don't miss it.
	if err := e.exec(`INSERT INTO unit_balances (unit_id) VALUES (?)`, uID); err != nil {
		t.Fatalf("seedUnit.balance %s: %v", number, err)
	}
	return uID
}

func (e *dashEnv) seedPerson(t *testing.T, bID uuid.UUID, phone, name string) uuid.UUID {
	t.Helper()
	pID := uuid.New()
	if err := e.exec(`INSERT INTO persons (id, building_id, full_name, phone) VALUES (?, ?, ?, ?)`, pID, bID, name, phone); err != nil {
		t.Fatalf("seedPerson: %v", err)
	}
	return pID
}

func (e *dashEnv) seedOccupancy(t *testing.T, unitID, personID uuid.UUID) {
	t.Helper()
	oID := uuid.New()
	if err := e.exec(`INSERT INTO occupancies (id, unit_id, person_id, relationship, start_date) VALUES (?, ?, ?, 'tenant', CURRENT_DATE)`, oID, unitID, personID); err != nil {
		t.Fatalf("seedOccupancy: %v", err)
	}
}

func (e *dashEnv) seedPeriod(t *testing.T, bID uuid.UUID, title string) uuid.UUID {
	t.Helper()
	pID := uuid.New()
	if err := e.exec(`INSERT INTO billing_periods (id, building_id, title, start_date, end_date, due_date, late_fee_type, late_fee_value, status) VALUES (?, ?, ?, CURRENT_DATE, CURRENT_DATE, CURRENT_DATE, 'none', 0, 'issued')`, pID, bID, title); err != nil {
		t.Fatalf("seedPeriod: %v", err)
	}
	return pID
}

func (e *dashEnv) seedInvoice(t *testing.T, bID, pID, unitID uuid.UUID, finalAmount, paidAmount int64, status, dueDate string) uuid.UUID {
	t.Helper()
	iID := uuid.New()
	invoiceNo := fmt.Sprintf("BLD-%s", iID.String()[:8])
	var dueVal any
	if dueDate != "" {
		dueVal = dueDate
	} else {
		dueVal = nil
	}
	if err := e.exec(`INSERT INTO invoices (id, invoice_number, building_id, period_id, unit_id, base_amount, prior_debt, late_fee_amount, credit_amount, final_amount, paid_amount, due_date, status) VALUES (?, ?, ?, ?, ?, ?, 0, 0, 0, ?, ?, ?, ?)`, iID, invoiceNo, bID, pID, unitID, finalAmount, finalAmount, paidAmount, dueVal, status); err != nil {
		t.Fatalf("seedInvoice: %v", err)
	}
	return iID
}

func (e *dashEnv) seedMaintenanceRequest(t *testing.T, bID uuid.UUID, unitID *uuid.UUID, residentID uuid.UUID, title, status, priority string) uuid.UUID {
	t.Helper()
	rID := uuid.New()
	if err := e.exec(`INSERT INTO maintenance_requests (id, building_id, unit_id, submitted_by, title, category, priority, status) VALUES (?, ?, ?, ?, ?, 'other', ?, ?)`, rID, bID, unitID, residentID, title, priority, status); err != nil {
		t.Fatalf("seedMaintenance: %v", err)
	}
	return rID
}

func (e *dashEnv) seedAnnouncement(t *testing.T, bID, mgrID uuid.UUID, title string, audienceType, audienceValue string, createdAt time.Time) uuid.UUID {
	t.Helper()
	aID := uuid.New()
	var av any
	if audienceValue != "" {
		av = audienceValue
	}
	publish := createdAt.Add(-time.Hour)
	expire := createdAt.Add(24 * time.Hour)
	if err := e.exec(`INSERT INTO announcements (id, building_id, title, body, audience_type, audience_value, publish_at, expire_at, created_by, created_at, updated_at) VALUES (?, ?, ?, '...', ?, ?, ?, ?, ?, ?, ?)`, aID, bID, title, audienceType, av, publish, expire, mgrID, createdAt, createdAt); err != nil {
		t.Fatalf("seedAnnouncement: %v", err)
	}
	return aID
}

func (e *dashEnv) markAnnouncementRead(t *testing.T, aID, userID uuid.UUID) {
	t.Helper()
	if err := e.exec(`INSERT INTO announcement_reads (announcement_id, user_id, read_at) VALUES (?, ?, ?)`, aID, userID, time.Now()); err != nil {
		t.Fatalf("markAnnouncementRead: %v", err)
	}
}

func (e *dashEnv) seedExpense(t *testing.T, bID, mgrID uuid.UUID, title string, amount int64, approval string, expenseDate string) uuid.UUID {
	t.Helper()
	eID := uuid.New()
	if err := e.exec(`INSERT INTO expenses (id, building_id, title, category, amount, expense_date, approval_status, created_by) VALUES (?, ?, ?, 'other', ?, ?, ?, ?)`, eID, bID, title, amount, expenseDate, approval, mgrID); err != nil {
		t.Fatalf("seedExpense: %v", err)
	}
	return eID
}

func (e *dashEnv) seedPayment(t *testing.T, bID, unitID uuid.UUID, amount int64, status string, paidAt string) uuid.UUID {
	t.Helper()
	pID := uuid.New()
	if err := e.exec(`INSERT INTO payments (id, building_id, unit_id, method, amount, paid_at, status) VALUES (?, ?, ?, 'manual', ?, ?, ?)`, pID, bID, unitID, amount, paidAt, status); err != nil {
		t.Fatalf("seedPayment: %v", err)
	}
	return pID
}

func (e *dashEnv) setUnitBalance(t *testing.T, unitID uuid.UUID, balance int64) {
	t.Helper()
	if err := e.exec(`UPDATE unit_balances SET balance = ? WHERE unit_id = ?`, balance, unitID); err != nil {
		t.Fatalf("setUnitBalance: %v", err)
	}
}

func (e *dashEnv) do(t *testing.T, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
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
		if out == nil {
			out = map[string]any{"_raw": w.Body.String()}
		}
	} else {
		out = map[string]any{}
	}
	return w.Code, out
}

func (e *dashEnv) get(t *testing.T, path, token string) (int, map[string]any) {
	return e.do(t, http.MethodGet, path, token, nil)
}

// --- US9 /me/home -----------------------------------------------------------

// T081 / Scenario 8.1 — A resident with one issued invoice and one open
// request must answer both "what do I owe / by when?" and "what happened to my
// request?" entirely from /me/home.
func TestResidentHomeAggregates(t *testing.T) {
	e := newDashEnv(t)

	mgrID, mgrTok := e.seedUser(t, "manager", "09120000001")
	bID := e.seedBuilding(t, mgrID)

	residentID, residentTok := e.seedUser(t, "resident", "09120000002")
	unitID := e.seedUnit(t, bID, "1", "occupied")
	personID := e.seedPerson(t, bID, "09120000002", "Ali Reza")
	e.seedOccupancy(t, unitID, personID)

	// One issued invoice (unpaid, $2,600,000 = 2_600_000 Toman).
	pID := e.seedPeriod(t, bID, "مهر ۱۴۰۵")
	due := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	e.seedInvoice(t, bID, pID, unitID, 2_600_000, 0, "unpaid", due)

	// One open maintenance request.
	e.seedMaintenanceRequest(t, bID, &unitID, residentID, "آسانسور لرزش دارد", "new", "urgent")
	now := time.Now()
	a1 := e.seedAnnouncement(t, bID, mgrID, "قطع آب فردا", "all", "", now.Add(-3*time.Hour))
	_ = e.seedAnnouncement(t, bID, mgrID, "جلسه هیئت‌مدیره", "all", "", now.Add(-2*time.Hour))
	_ = e.seedAnnouncement(t, bID, mgrID, "تعمیر پشت‌بام", "all", "", now.Add(-1*time.Hour))
	e.markAnnouncementRead(t, a1, residentID)

	code, body := e.get(t, "/api/v1/me/home", residentTok)
	if code != http.StatusOK {
		t.Fatalf("resident /me/home: code=%d body=%v", code, body)
	}

	// Payable amount = sum of unpaid/partial/expired final_amount - paid_amount.
	if got, _ := body["payable_amount"].(float64); got != 2_600_000 {
		t.Fatalf("payable_amount: want 2_600_000 got %v", body["payable_amount"])
	}
	if got, _ := body["unit_count"].(float64); got != 1 {
		t.Fatalf("unit_count: want 1 got %v", body["unit_count"])
	}
	// Open request count.
	if got, _ := body["open_request_count"].(float64); got != 1 {
		t.Fatalf("open_request_count: want 1 got %v", body["open_request_count"])
	}

	// Latest invoice.
	li, ok := body["latest_invoice"].(map[string]any)
	if !ok || li == nil {
		t.Fatalf("latest_invoice missing: %v", body["latest_invoice"])
	}
	if got, _ := li["final_amount"].(float64); got != 2_600_000 {
		t.Fatalf("latest_invoice.final_amount: want 2_600_000 got %v", li["final_amount"])
	}
	if got, _ := li["status"].(string); got != "unpaid" {
		t.Fatalf("latest_invoice.status: want unpaid got %v", li["status"])
	}
	if got, _ := li["due_date"].(string); got != due {
		t.Fatalf("latest_invoice.due_date: want %s got %v", due, li["due_date"])
	}

	// Latest request — same id as the seeded one.
	lr, ok := body["latest_request"].(map[string]any)
	if !ok || lr == nil {
		t.Fatalf("latest_request missing: %v", body["latest_request"])
	}
	if got, _ := lr["title"].(string); got != "آسانسور لرزش دارد" {
		t.Fatalf("latest_request.title: want 'آسانسور لرزش دارد' got %v", lr["title"])
	}
	if got, _ := lr["priority"].(string); got != "urgent" {
		t.Fatalf("latest_request.priority: want urgent got %v", lr["priority"])
	}
	if got, _ := lr["status"].(string); got != "new" {
		t.Fatalf("latest_request.status: want new got %v", lr["status"])
	}

	// Announcements — newest 5 + unread count = 2 (a2 unread, a3 unread).
	las, _ := body["latest_announcements"].([]any)
	if len(las) != 3 {
		t.Fatalf("latest_announcements length: want 3 got %d", len(las))
	}
	// Newest first.
	if got, _ := las[0].(map[string]any)["title"].(string); got != "تعمیر پشت‌بام" {
		t.Fatalf("latest_announcements[0].title: want 'تعمیر پشت‌بام' got %v", las[0])
	}
	if got, _ := body["unread_announcement_count"].(float64); got != 2 {
		t.Fatalf("unread_announcement_count: want 2 got %v", body["unread_announcement_count"])
	}
	// a1 was marked read; the others should be unread.
	readCount := 0
	for _, x := range las {
		m := x.(map[string]any)
		if m["is_read"] == true {
			readCount++
		}
	}
	if readCount != 1 {
		t.Fatalf("latest_announcements read flag: want 1 read got %d (titles=%v)", readCount, las)
	}

	// Manager calling /me/home: empty sections, no errors. Manager has no
	// active occupancies, so the endpoint returns a well-formed zero state.
	code, body = e.get(t, "/api/v1/me/home", mgrTok)
	if code != http.StatusOK {
		t.Fatalf("manager /me/home: code=%d body=%v", code, body)
	}
	if got, _ := body["payable_amount"].(float64); got != 0 {
		t.Fatalf("manager payable_amount: want 0 got %v", body["payable_amount"])
	}
	if got, _ := body["unit_count"].(float64); got != 0 {
		t.Fatalf("manager unit_count: want 0 got %v", body["unit_count"])
	}
	if li, _ := body["latest_invoice"].(map[string]any); li != nil {
		t.Fatalf("manager latest_invoice: want nil got %v", li)
	}
}

// T081 — Resident isolation: a resident's /me/home never reflects another
// resident's invoices or requests, even when the DB contains them.
func TestResidentHomeIsolation(t *testing.T) {
	e := newDashEnv(t)

	mgrID, _ := e.seedUser(t, "manager", "09120000001")
	bID := e.seedBuilding(t, mgrID)

	residentA, tokA := e.seedUser(t, "resident", "09120000010")
	residentB, _ := e.seedUser(t, "resident", "09120000011")

	unitA := e.seedUnit(t, bID, "1", "occupied")
	unitB := e.seedUnit(t, bID, "2", "occupied")

	personA := e.seedPerson(t, bID, "09120000010", "A")
	personB := e.seedPerson(t, bID, "09120000011", "B")
	e.seedOccupancy(t, unitA, personA)
	e.seedOccupancy(t, unitB, personB)

	pID := e.seedPeriod(t, bID, "مهر ۱۴۰۵")
	due := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	// Resident B's invoice (1_000_000) — should NOT show on resident A's home.
	e.seedInvoice(t, bID, pID, unitB, 1_000_000, 0, "unpaid", due)
	// Resident A's invoice (5_000_000) — should show on resident A's home.
	e.seedInvoice(t, bID, pID, unitA, 5_000_000, 0, "unpaid", due)

	// Resident B's request — should NOT show on resident A's home.
	e.seedMaintenanceRequest(t, bID, &unitB, residentB, "نشتی آب", "new", "important")
	e.seedMaintenanceRequest(t, bID, &unitA, residentA, "خرابی شیر آب", "new", "normal")

	code, body := e.get(t, "/api/v1/me/home", tokA)
	if code != http.StatusOK {
		t.Fatalf("resident A /me/home: code=%d body=%v", code, body)
	}

	if got, _ := body["payable_amount"].(float64); got != 5_000_000 {
		t.Fatalf("resident A payable_amount: want 5_000_000 (own invoice only) got %v", body["payable_amount"])
	}
	lr, _ := body["latest_request"].(map[string]any)
	if lr == nil {
		t.Fatalf("resident A latest_request: nil")
	}
	if got, _ := lr["title"].(string); got != "خرابی شیر آب" {
		t.Fatalf("resident A latest_request.title: want 'خرابی شیر آب' got %v", lr["title"])
	}
}

// T081 — Unit scoped: a resident occupying multiple units sees the union
// (payable sum across all units, latest invoice = newest by issue date).
func TestResidentHomeMultipleUnits(t *testing.T) {
	e := newDashEnv(t)

	mgrID, _ := e.seedUser(t, "manager", "09120000001")
	bID := e.seedBuilding(t, mgrID)

	_, tok := e.seedUser(t, "resident", "09120000020")

	u1 := e.seedUnit(t, bID, "1", "occupied")
	u2 := e.seedUnit(t, bID, "2", "occupied")

	personID := e.seedPerson(t, bID, "09120000020", "Big Family")
	e.seedOccupancy(t, u1, personID)
	e.seedOccupancy(t, u2, personID)

	pID := e.seedPeriod(t, bID, "مهر ۱۴۰۵")
	due := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	e.seedInvoice(t, bID, pID, u1, 2_600_000, 0, "unpaid", due)
	// Slightly newer invoice on u2.
	_ = e.seedInvoice(t, bID, pID, u2, 1_400_000, 0, "unpaid", time.Now().Add(72*time.Hour).Format("2006-01-02"))

	code, body := e.get(t, "/api/v1/me/home", tok)
	if code != http.StatusOK {
		t.Fatalf("/me/home: code=%d body=%v", code, body)
	}
	if got, _ := body["payable_amount"].(float64); got != 4_000_000 {
		t.Fatalf("payable_amount: want 4_000_000 got %v", body["payable_amount"])
	}
	if got, _ := body["unit_count"].(float64); got != 2 {
		t.Fatalf("unit_count: want 2 got %v", body["unit_count"])
	}
}

// --- US10 /buildings/{id}/dashboard ----------------------------------------

// T085 / Scenario 8.2 — With 7 debtor units and 3 open requests seeded, the
// dashboard cards and alerts show exactly those numbers (FR-035, SC-007).
func TestManagerDashboardAggregation(t *testing.T) {
	e := newDashEnv(t)

	mgrID, mgrTok := e.seedUser(t, "manager", "09120000001")
	bID := e.seedBuilding(t, mgrID)

	residentID, _ := e.seedUser(t, "resident", "09120000030")
	personID := e.seedPerson(t, bID, "09120000030", "R")

	// 10 units total: 7 debtor (positive unit_balances.balance), 3 paid off.
	var firstUnit uuid.UUID
	for i := 1; i <= 10; i++ {
		uID := e.seedUnit(t, bID, fmt.Sprintf("%d", i), "occupied")
		if i == 1 {
			firstUnit = uID
			e.seedOccupancy(t, uID, personID)
		}
		if i <= 7 {
			e.setUnitBalance(t, uID, int64(i*100_000))
		}
	}

	// 3 open maintenance requests.
	for i := 1; i <= 3; i++ {
		e.seedMaintenanceRequest(t, bID, nil, residentID,
			fmt.Sprintf("درخواست %d", i), "new", "normal")
	}
	// 1 closed request — must NOT count as open.
	e.seedMaintenanceRequest(t, bID, nil, residentID, "قدیمی", "closed", "normal")

	// Month income + expense anchor (today's month).
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthDate := monthStart.Format("2006-01-02")
	e.seedPayment(t, bID, firstUnit, 3_000_000, "verified", monthDate)
	e.seedExpense(t, bID, mgrID, "برق لابی", 1_500_000, "approved", monthDate)
	e.seedExpense(t, bID, mgrID, "نگهبانی", 2_500_000, "pending", monthDate)

	code, body := e.get(t, "/api/v1/buildings/"+bID.String()+"/dashboard", mgrTok)
	if code != http.StatusOK {
		t.Fatalf("manager dashboard: code=%d body=%v", code, body)
	}

	if got, _ := body["building_id"].(string); got != bID.String() {
		t.Fatalf("building_id: want %s got %v", bID, body["building_id"])
	}
	if got, _ := body["unit_count"].(float64); got != 10 {
		t.Fatalf("unit_count: want 10 got %v", body["unit_count"])
	}
	if got, _ := body["debtor_unit_count"].(float64); got != 7 {
		t.Fatalf("debtor_unit_count: want 7 got %v", body["debtor_unit_count"])
	}
	if got, _ := body["total_debt"].(float64); got != 2_800_000 { // 100+200+...+700
		t.Fatalf("total_debt: want 2_800_000 got %v", body["total_debt"])
	}
	if got, _ := body["month_income"].(float64); got != 3_000_000 {
		t.Fatalf("month_income: want 3_000_000 got %v", body["month_income"])
	}
	if got, _ := body["month_expense"].(float64); got != 4_000_000 { // approved + pending
		t.Fatalf("month_expense: want 4_000_000 got %v", body["month_expense"])
	}
	if got, _ := body["open_requests"].(float64); got != 3 {
		t.Fatalf("open_requests: want 3 got %v", body["open_requests"])
	}
	if got, _ := body["pending_expenses"].(float64); got != 1 {
		t.Fatalf("pending_expenses: want 1 got %v", body["pending_expenses"])
	}

	// Alert composition: 7 debtor + 3 open request + 1 pending expense = 11.
	alerts, _ := body["alerts"].([]any)
	if len(alerts) != 11 {
		t.Fatalf("alerts length: want 11 got %d (%v)", len(alerts), alerts)
	}
	counts := map[string]int{}
	for _, a := range alerts {
		m := a.(map[string]any)
		counts[m["kind"].(string)]++
	}
	if counts["debtor_unit"] != 7 {
		t.Fatalf("debtor_unit alerts: want 7 got %d", counts["debtor_unit"])
	}
	if counts["open_request"] != 3 {
		t.Fatalf("open_request alerts: want 3 got %d", counts["open_request"])
	}
	if counts["pending_expense"] != 1 {
		t.Fatalf("pending_expense alerts: want 1 got %d", counts["pending_expense"])
	}

	// Quick actions: 5 keys, every path deep-links to /manager/...
	qas, _ := body["quick_actions"].([]any)
	if len(qas) != 5 {
		t.Fatalf("quick_actions length: want 5 got %d", len(qas))
	}
	for _, q := range qas {
		m := q.(map[string]any)
		if path, _ := m["path"].(string); !strings.HasPrefix(path, "/manager/") {
			t.Fatalf("quick_action path: %s must start with /manager/", path)
		}
	}
}

// T085 — Manager scope: non-granted manager → 403; resident → 403.
func TestManagerDashboardScope(t *testing.T) {
	e := newDashEnv(t)

	grantedMgrID, grantedMgrTok := e.seedUser(t, "manager", "09120000001")
	bID := e.seedBuilding(t, grantedMgrID)

	// A second manager who holds no user_buildings grant for this building.
	otherMgrID, otherMgrTok := e.seedUser(t, "manager", "09120000002")
	_ = otherMgrID

	// Resident — must never be able to call the manager route.
	_, residentTok := e.seedUser(t, "resident", "09120000003")

	// 1. Resident → 403.
	code, _ := e.get(t, "/api/v1/buildings/"+bID.String()+"/dashboard", residentTok)
	if code != http.StatusForbidden {
		t.Fatalf("resident on manager dashboard: want 403 got %d", code)
	}

	// 2. Non-granted manager → 403.
	_ = otherMgrID
	code, _ = e.get(t, "/api/v1/buildings/"+bID.String()+"/dashboard", otherMgrTok)
	if code != http.StatusForbidden {
		t.Fatalf("non-granted manager on dashboard: want 403 got %d", code)
	}

	// 3. Granted manager → 200.
	code, body := e.get(t, "/api/v1/buildings/"+bID.String()+"/dashboard", grantedMgrTok)
	if code != http.StatusOK {
		t.Fatalf("granted manager: want 200 got %d body=%v", code, body)
	}
}

// T085 — Past-due invoices surface in the alerts list with severity 2 and
// the correct outstanding amount.
func TestManagerDashboardPastDueAlerts(t *testing.T) {
	e := newDashEnv(t)

	mgrID, mgrTok := e.seedUser(t, "manager", "09120000001")
	bID := e.seedBuilding(t, mgrID)

	unitID := e.seedUnit(t, bID, "1", "occupied")
	pID := e.seedPeriod(t, bID, "مهر ۱۴۰۵")

	// Past-due: due yesterday, partial paid.
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	e.seedInvoice(t, bID, pID, unitID, 2_600_000, 500_000, "partial", yesterday)
	// Not due yet — must NOT appear as past-due.
	future := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	e.seedInvoice(t, bID, pID, unitID, 1_000_000, 0, "unpaid", future)

	code, body := e.get(t, "/api/v1/buildings/"+bID.String()+"/dashboard", mgrTok)
	if code != http.StatusOK {
		t.Fatalf("dashboard: code=%d body=%v", code, body)
	}

	alerts, _ := body["alerts"].([]any)
	var pastDueAlerts []map[string]any
	for _, a := range alerts {
		m := a.(map[string]any)
		if m["kind"] == "past_due_invoice" {
			pastDueAlerts = append(pastDueAlerts, m)
		}
	}
	if len(pastDueAlerts) != 1 {
		t.Fatalf("past_due_invoice alerts: want 1 got %d (%v)", len(pastDueAlerts), alerts)
	}
	if got, _ := pastDueAlerts[0]["amount"].(float64); got != 2_100_000 { // 2_600_000 - 500_000
		t.Fatalf("past_due amount: want 2_100_000 got %v", pastDueAlerts[0]["amount"])
	}
	if got, _ := pastDueAlerts[0]["severity"].(float64); got != 2 {
		t.Fatalf("past_due severity: want 2 (danger) got %v", pastDueAlerts[0]["severity"])
	}
}
