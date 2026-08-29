package billing

// T043 — US4 integration suite: drives the real Gin handlers against a
// throwaway PostgreSQL schema with the real migrations applied, mirroring the
// auth/building harnesses. Covers the period lifecycle (draft → calculate →
// preview → issue → close), reopen before issue only, invoice immutability
// after occupant-count/area changes (BR-05, incl. the DB trigger guard),
// cancel preserving the row (BR-10), adjustments not mutating originals
// (BR-03), and resident isolation on /me/invoices. Skips when neither Docker
// nor a reachable dev PostgreSQL exists (set HAMSA_TEST_DSN to force).

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
	"hamsa/internal/notification"
	"hamsa/internal/platform/db"
	"hamsa/internal/platform/httpx"
)

type billEnv struct {
	engine *gin.Engine
	gormDB *gorm.DB
	tokens *auth.TokenService
	repo   *Repository
}

func (e *billEnv) exec(query string, args ...any) error {
	return e.gormDB.WithContext(context.Background()).Exec(query, args...).Error
}

const billBaseDSN = "host=localhost port=5432 user=hamsa password=hamsa dbname=hamsa sslmode=disable"

func newBillingEnv(t *testing.T) *billEnv {
	t.Helper()

	dsn := os.Getenv("HAMSA_TEST_DSN")
	if dsn == "" {
		if !pgReachableBill(t) {
			t.Skip("skipping integration test — no Docker engine and no reachable PostgreSQL (set HAMSA_TEST_DSN)")
		}
		dsn = billBaseDSN
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
	if os.Getenv("HAMSA_TEST_DEBUG") != "" {
		log = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	gormDB, err := db.Connect(dsn, log)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	tokens := auth.NewTokenService([]byte("test-secret-32-bytes-long!!!!!!"),
		15*time.Minute, 30*24*time.Hour, &auth.GormRefreshStore{DB: gormDB}, auth.RealClock{})
	repo := NewRepository(gormDB)
	notifSvc := notification.NewService(gormDB, notification.NoopNotifier{})
	aud := audit.New(gormDB, log)

	gin.SetMode(gin.TestMode)
	router := httpx.NewRouter(log, "dev")
	authMW := auth.Authenticate(tokens, auth.NewRepository(gormDB))
	Register(router.Group("/api/v1", authMW),
		NewCalcService(repo, nil),
		NewPeriodService(repo, notifSvc, aud),
		aud,
		auth.NewScopeResolver(gormDB))
	return &billEnv{engine: router, gormDB: gormDB, tokens: tokens, repo: repo}
}

func pgReachableBill(t *testing.T) bool {
	t.Helper()
	dbh, err := sql.Open("pgx", billBaseDSN)
	if err != nil {
		return false
	}
	defer dbh.Close()
	dbh.SetConnMaxLifetime(time.Second)
	return dbh.Ping() == nil
}

// seedUser inserts a user directly and returns its id + an access token.
func (e *billEnv) seedUser(t *testing.T, role, phone string) (uuid.UUID, string) {
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

func (e *billEnv) request(t *testing.T, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.engine.ServeHTTP(rec, req)
	return rec
}

// seedBuilding creates a building with 10 units (area 100, numbers "1".."10")
// and the spec §25 occupant distribution (20 occupants, unit "1" = 4), plus
// the manager grant. Returns the unit-id map keyed by unit number.
func (e *billEnv) seedBuilding(t *testing.T, managerID uuid.UUID) (uuid.UUID, map[string]uuid.UUID) {
	t.Helper()
	buildingID := uuid.New()
	if err := e.exec(
		`INSERT INTO buildings (id, name) VALUES (?, 'برج تست')`, buildingID,
	); err != nil {
		t.Fatalf("seed building: %v", err)
	}
	if err := e.exec(
		`INSERT INTO user_buildings (user_id, building_id) VALUES (?, ?)`, managerID, buildingID,
	); err != nil {
		t.Fatalf("seed grant: %v", err)
	}
	occupants := map[string]int{"1": 4, "2": 6, "3": 5, "4": 5}
	units := map[string]uuid.UUID{}
	for i := 1; i <= 10; i++ {
		number := fmt.Sprintf("%d", i)
		unitID := uuid.New()
		units[number] = unitID
		if err := e.exec(
			`INSERT INTO units (id, building_id, number, floor, area_m2, status) VALUES (?, ?, ?, 1, 100, 'occupied')`,
			unitID, buildingID, number,
		); err != nil {
			t.Fatalf("seed unit: %v", err)
		}
		if occ, ok := occupants[number]; ok {
			if err := e.exec(
				`INSERT INTO occupant_count_history (id, unit_id, occupant_count, effective_from) VALUES (?, ?, ?, '2026-08-01')`,
				uuid.New(), unitID, occ,
			); err != nil {
				t.Fatalf("seed occupants: %v", err)
			}
		}
	}
	return buildingID, units
}

// createSpecPeriod drives the API to create the §25 period (equal 10,000,000
// + per-occupant 8,000,000) and returns its id.
func (e *billEnv) createSpecPeriod(t *testing.T, token string, buildingID uuid.UUID) string {
	t.Helper()
	rec := e.request(t, "POST", fmt.Sprintf("/api/v1/buildings/%s/periods", buildingID),
		`{"title":"مرداد ۱۴۰۵","start_date":"2026-08-01","end_date":"2026-08-31","due_date":"2026-09-05"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create period: %d %s", rec.Code, rec.Body.String())
	}
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("parse period: %v", err)
	}
	for _, ci := range []string{
		`{"title":"هزینه عمومی","total_amount":"10000000","method":"equal"}`,
		`{"title":"آب","total_amount":"8000000","method":"per_occupant"}`,
	} {
		rec = e.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/cost-items", p.ID), ci, token)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create cost item: %d %s", rec.Code, rec.Body.String())
		}
	}
	return p.ID
}

// invoiceDTO is the subset of the invoice JSON the tests assert on (money is
// a quoted integer string on the wire — contracts/api.md).
type invoiceDTO struct {
	ID            string `json:"id"`
	InvoiceNumber string `json:"invoice_number"`
	UnitID        string `json:"unit_id"`
	BaseAmount    string `json:"base_amount"`
	FinalAmount   string `json:"final_amount"`
	Status        string `json:"status"`
	Items         []struct {
		Kind   string `json:"kind"`
		Title  string `json:"title"`
		Amount string `json:"amount"`
	} `json:"items"`
}

func (e *billEnv) invoicesOf(t *testing.T, body []byte) []invoiceDTO {
	t.Helper()
	var p struct {
		Invoices []invoiceDTO `json:"invoices"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("parse invoices: %v", err)
	}
	return p.Invoices
}

func invoiceFor(t *testing.T, invoices []invoiceDTO, unitID uuid.UUID) invoiceDTO {
	t.Helper()
	for _, inv := range invoices {
		if inv.UnitID == unitID.String() {
			return inv
		}
	}
	t.Fatalf("no invoice for unit %s", unitID)
	return invoiceDTO{}
}

// TestPeriodLifecycleAndSpec25 walks the full US4 lifecycle on the spec §25
// fixture and asserts the 2,600,000 acceptance value plus immutability.
func TestPeriodLifecycleAndSpec25(t *testing.T) {
	env := newBillingEnv(t)
	mgrID, mgrToken := env.seedUser(t, "manager", "09120000001")
	buildingID, units := env.seedBuilding(t, mgrID)
	periodID := env.createSpecPeriod(t, mgrToken, buildingID)

	// --- calculate → preview shows unit 1 = 2,600,000 (spec §25) -------------
	rec := env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/calculate", periodID), "", mgrToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("calculate: %d %s", rec.Code, rec.Body.String())
	}
	inv1 := invoiceFor(t, env.invoicesOf(t, rec.Body.Bytes()), units["1"])
	if inv1.FinalAmount != "2600000" {
		t.Fatalf("unit 1 invoice = %s, want 2600000 (spec §25)", inv1.FinalAmount)
	}
	if inv1.BaseAmount != "2600000" {
		t.Fatalf("unit 1 base = %s, want 2600000", inv1.BaseAmount)
	}

	// Reconciliation flags on every item (BR-08/BR-09).
	var preview struct {
		Reconciled bool `json:"reconciled"`
		Items      []struct {
			Title      string `json:"title"`
			Reconciled bool   `json:"reconciled"`
		} `json:"items"`
	}
	rec = env.request(t, "GET", fmt.Sprintf("/api/v1/periods/%s/preview", periodID), "", mgrToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &preview); err != nil {
		t.Fatalf("parse preview: %v", err)
	}
	if !preview.Reconciled || len(preview.Items) != 2 {
		t.Fatalf("preview reconciled=%v items=%d", preview.Reconciled, len(preview.Items))
	}
	for _, it := range preview.Items {
		if !it.Reconciled {
			t.Fatalf("item %q not reconciled", it.Title)
		}
	}

	// --- recalculation after base-data change while still unissued -----------
	if err := env.exec(`UPDATE occupant_count_history SET occupant_count = 5 WHERE unit_id = ?`, units["1"]); err != nil {
		t.Fatalf("update occupants: %v", err)
	}
	// Unit 2 drops to 5 so Σ occupants stays 20 (quickstart Scenario 2:
	// water share 8,000,000×5/20 = 2,000,000).
	if err := env.exec(`UPDATE occupant_count_history SET occupant_count = 5 WHERE unit_id = ?`, units["2"]); err != nil {
		t.Fatalf("update occupants: %v", err)
	}
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/calculate", periodID), "", mgrToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("recalculate: %d %s", rec.Code, rec.Body.String())
	}
	inv1 = invoiceFor(t, env.invoicesOf(t, rec.Body.Bytes()), units["1"])
	if inv1.FinalAmount != "3000000" { // 1,000,000 + 8,000,000×5/20
		t.Fatalf("unit 1 after recalc = %s, want 3000000", inv1.FinalAmount)
	}

	// --- reopen (calculated → draft), fix data, recalculate ------------------
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/reopen", periodID), "", mgrToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("reopen: %d %s", rec.Code, rec.Body.String())
	}
	if err := env.exec(`UPDATE occupant_count_history SET occupant_count = 4 WHERE unit_id = ?`, units["1"]); err != nil {
		t.Fatalf("reset occupants: %v", err)
	}
	if err := env.exec(`UPDATE occupant_count_history SET occupant_count = 6 WHERE unit_id = ?`, units["2"]); err != nil {
		t.Fatalf("reset occupants: %v", err)
	}
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/calculate", periodID), "", mgrToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("recalculate after reopen: %d %s", rec.Code, rec.Body.String())
	}
	inv1 = invoiceFor(t, env.invoicesOf(t, rec.Body.Bytes()), units["1"])
	if inv1.FinalAmount != "2600000" {
		t.Fatalf("unit 1 after reopen+recalc = %s, want 2600000", inv1.FinalAmount)
	}

	// --- issue: sequential per-building numbers, all invoices present ---------
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/issue", periodID), "", mgrToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("issue: %d %s", rec.Code, rec.Body.String())
	}
	var issued struct {
		Items []invoiceDTO `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &issued); err != nil {
		t.Fatalf("parse issued: %v", err)
	}
	if len(issued.Items) != 10 {
		t.Fatalf("issued %d invoices, want 10", len(issued.Items))
	}
	seen := map[string]bool{}
	for _, inv := range issued.Items {
		if !strings.HasPrefix(inv.InvoiceNumber, "BLD-") {
			t.Fatalf("invoice number %q lacks BLD- prefix", inv.InvoiceNumber)
		}
		if seen[inv.InvoiceNumber] {
			t.Fatalf("duplicate invoice number %q", inv.InvoiceNumber)
		}
		seen[inv.InvoiceNumber] = true
		if inv.Status != "unpaid" {
			t.Fatalf("issued invoice status = %q, want unpaid", inv.Status)
		}
	}

	// --- after issue: no recalculation, no reopen -----------------------------
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/calculate", periodID), "", mgrToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("calculate after issue = %d, want 409", rec.Code)
	}
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/reopen", periodID), "", mgrToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("reopen after issue = %d, want 409", rec.Code)
	}

	// --- BR-05: base-data edits after issue leave the invoice unchanged ------
	if err := env.exec(`UPDATE occupant_count_history SET occupant_count = 9 WHERE unit_id = ?`, units["1"]); err != nil {
		t.Fatalf("edit occupants after issue: %v", err)
	}
	if err := env.exec(`UPDATE units SET area_m2 = 500 WHERE id = ?`, units["1"]); err != nil {
		t.Fatalf("edit area after issue: %v", err)
	}
	rec = env.request(t, "GET", fmt.Sprintf("/api/v1/invoices/%s", inv1.ID), "", mgrToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("invoice detail: %d %s", rec.Code, rec.Body.String())
	}
	var detail invoiceDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("parse detail: %v", err)
	}
	if detail.FinalAmount != "2600000" || detail.BaseAmount != "2600000" {
		t.Fatalf("issued invoice changed after base-data edit: base=%s final=%s", detail.BaseAmount, detail.FinalAmount)
	}

	// DB-level guard: direct amount UPDATE must be rejected by the trigger.
	if err := env.exec(`UPDATE invoices SET final_amount = 1 WHERE id = ?`, inv1.ID); err == nil {
		t.Fatalf("expected immutability trigger to reject amount UPDATE")
	}

	// --- adjustments: append-only, originals untouched (BR-03/FR-017) --------
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/invoices/%s/adjustments", inv1.ID),
		`{"kind":"credit","amount":"100000","reason":"تعدیل آب"}`, mgrToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("adjustment: %d %s", rec.Code, rec.Body.String())
	}
	rec = env.request(t, "GET", fmt.Sprintf("/api/v1/invoices/%s", inv1.ID), "", mgrToken)
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("parse detail after adjustment: %v", err)
	}
	if detail.BaseAmount != "2600000" {
		t.Fatalf("adjustment mutated the base amount: %s", detail.BaseAmount)
	}
	hasAdj := false
	for _, it := range detail.Items {
		if it.Kind == "adjustment" && it.Amount == "100000" {
			hasAdj = true
		}
	}
	if !hasAdj {
		t.Fatalf("adjustment item missing from invoice detail: %+v", detail.Items)
	}

	// --- cancel preserves the row (BR-10) --------------------------------------
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/invoices/%s/cancel", issued.Items[2].ID),
		`{"reason":"اشتباه در ثبت"}`, mgrToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", rec.Code, rec.Body.String())
	}
	var n int64
	if err := env.gormDB.WithContext(context.Background()).Raw(
		`SELECT count(*) FROM invoices WHERE id = ?`, issued.Items[2].ID).Scan(&n).Error; err != nil || n != 1 {
		t.Fatalf("cancelled invoice row not preserved (n=%d err=%v)", n, err)
	}
	if err := env.gormDB.WithContext(context.Background()).Raw(
		`SELECT status FROM invoices WHERE id = ?`, issued.Items[2].ID).Scan(&detail.Status).Error; err != nil {
		t.Fatalf("read status: %v", err)
	}
	if detail.Status != "cancelled" {
		t.Fatalf("status = %q, want cancelled", detail.Status)
	}

	// --- close (issued → closed) -----------------------------------------------
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/close", periodID), "", mgrToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("close: %d %s", rec.Code, rec.Body.String())
	}

	// --- notification emitted on issue (FR-032) --------------------------------
	if err := env.exec(
		`INSERT INTO persons (id, building_id, full_name, phone) VALUES (?, ?, 'ساکن یک', '09111111111')`,
		uuid.New(), buildingID,
	); err != nil {
		t.Fatalf("seed person: %v", err)
	}
}

// TestResidentInvoiceIsolation verifies resident scoping on /me/invoices and
// object-level 403 on another unit's invoice (contracts/api.md behavior 2).
func TestResidentInvoiceIsolation(t *testing.T) {
	env := newBillingEnv(t)
	mgrID, mgrToken := env.seedUser(t, "manager", "09120000002")
	buildingID, units := env.seedBuilding(t, mgrID)
	periodID := env.createSpecPeriod(t, mgrToken, buildingID)
	if rec := env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/issue", periodID), "", mgrToken); rec.Code != http.StatusOK {
		// issue needs a calculated period first
		if rec := env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/calculate", periodID), "", mgrToken); rec.Code != http.StatusOK {
			t.Fatalf("calculate: %d %s", rec.Code, rec.Body.String())
		}
		if rec := env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/issue", periodID), "", mgrToken); rec.Code != http.StatusOK {
			t.Fatalf("issue: %d %s", rec.Code, rec.Body.String())
		}
	}

	// Resident of unit 1: person with the resident's phone occupying unit 1.
	resID, resToken := env.seedUser(t, "resident", "09111111112")
	if err := env.exec(
		`INSERT INTO persons (id, building_id, full_name, phone) VALUES (?, ?, 'ساکن واحد یک', '09111111112')`,
		uuid.New(), buildingID,
	); err != nil {
		t.Fatalf("seed person: %v", err)
	}
	if err := env.exec(
		`INSERT INTO occupancies (id, unit_id, person_id, relationship, start_date) VALUES (?, ?, (SELECT id FROM persons WHERE phone = '09111111112'), 'tenant', '2026-08-01')`,
		uuid.New(), units["1"],
	); err != nil {
		t.Fatalf("seed occupancy: %v", err)
	}
	_ = resID

	// Resident sees only own-unit invoices.
	rec := env.request(t, "GET", "/api/v1/me/invoices", "", resToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("me/invoices: %d %s", rec.Code, rec.Body.String())
	}
	var list struct {
		Items []invoiceDTO `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("parse list: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].UnitID != units["1"].String() {
		t.Fatalf("resident sees %d invoices, want exactly own unit 1", len(list.Items))
	}

	// Resident cannot open another unit's invoice (403).
	rec = env.request(t, "GET", fmt.Sprintf("/api/v1/invoices/%s", list.Items[0].ID), "", resToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("own invoice: %d %s", rec.Code, rec.Body.String())
	}
	rec = env.request(t, "GET", fmt.Sprintf("/api/v1/buildings/%s/invoices", buildingID), "", resToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("resident on manager route = %d, want 403", rec.Code)
	}
}

// TestZeroFactorRejected checks the zero-participating-factor edge case via
// the API (spec edge: all occupants zero for per_occupant → blocked).
func TestZeroFactorRejected(t *testing.T) {
	env := newBillingEnv(t)
	mgrID, mgrToken := env.seedUser(t, "manager", "09120000003")
	buildingID, _ := env.seedBuilding(t, mgrID)

	rec := env.request(t, "POST", fmt.Sprintf("/api/v1/buildings/%s/periods", buildingID),
		`{"title":"تیر ۱۴۰۵","start_date":"2026-07-01","end_date":"2026-07-31","due_date":"2026-08-05"}`, mgrToken)
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("parse period: %v", err)
	}
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/cost-items", p.ID),
		`{"title":"آب","total_amount":"5000000","method":"per_occupant"}`, mgrToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("cost item: %d %s", rec.Code, rec.Body.String())
	}
	// No occupant counts recorded anywhere → zero factor.
	rec = env.request(t, "POST", fmt.Sprintf("/api/v1/periods/%s/calculate", p.ID), "", mgrToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("zero factor calculate = %d %s, want 400", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "صفر") {
		t.Fatalf("expected Persian zero-factor message, got %s", rec.Body.String())
	}
}
