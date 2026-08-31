package maintenance

// T068 — US7 integration suite: drives the real Gin handlers against a
// throwaway PostgreSQL schema with the real migrations applied (same harness
// pattern as expense/billing/building). Covers contracts/api.md
// "Maintenance" and quickstart Scenario 6:
//
//   - resident submit (new) + own-list isolation + manager list scope
//   - legal state transitions only: 409 on backwards/illegal skips,
//     early-close allowed from under_review/in_progress/done, terminal closed
//   - assignee/cost/notes updates with validation (400 on bad person/cost)
//   - notifications emitted per transition (to the submitter) + audit entries
//   - photo file binding via the T010 files registry
//   - resident isolation: a resident sees only own requests (403 on чужой id)
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

type maintEnv struct {
	engine *gin.Engine
	gormDB *gorm.DB
	tokens *auth.TokenService
}

func (e *maintEnv) exec(query string, args ...any) error {
	return e.gormDB.WithContext(context.Background()).Exec(query, args...).Error
}

const maintBaseDSN = "host=localhost port=5432 user=hamsa password=hamsa dbname=hamsa sslmode=disable"

func newMaintEnv(t *testing.T) *maintEnv {
	t.Helper()
	dsn := os.Getenv("HAMSA_TEST_DSN")
	if dsn == "" {
		if !pgReachableMaint(t) {
			t.Skip("skipping integration test — no Docker engine and no reachable PostgreSQL (set HAMSA_TEST_DSN)")
		}
		dsn = maintBaseDSN
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
	Register(authed, NewService(NewRepository(gormDB), notification.NewService(gormDB, notification.NoopNotifier{}), audit.New(gormDB, log)), audit.New(gormDB, log))
	return &maintEnv{engine: router, gormDB: gormDB, tokens: tokens}
}

func pgReachableMaint(t *testing.T) bool {
	t.Helper()
	dbh, err := sql.Open("pgx", maintBaseDSN)
	if err != nil {
		return false
	}
	defer dbh.Close()
	dbh.SetConnMaxLifetime(time.Second)
	return dbh.Ping() == nil
}

func (e *maintEnv) seedUser(t *testing.T, role, phone string) (uuid.UUID, string) {
	t.Helper()
	id := uuid.New()
	if err := e.exec(`INSERT INTO users (id, phone, role, name) VALUES (?, ?, ?, ?)`, id, phone, role, "تست "+role); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	token, _, err := e.tokens.IssueAccessToken(id)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return id, token
}

func (e *maintEnv) seedBuilding(t *testing.T, mgrID uuid.UUID) uuid.UUID {
	t.Helper()
	bID := uuid.New()
	if err := e.exec(`INSERT INTO buildings (id, name) VALUES (?, 'برج نگهداری')`, bID); err != nil {
		t.Fatalf("seed building: %v", err)
	}
	if err := e.exec(`INSERT INTO user_buildings (user_id, building_id) VALUES (?, ?)`, mgrID, bID); err != nil {
		t.Fatalf("seed grant: %v", err)
	}
	return bID
}

func (e *maintEnv) seedUnit(t *testing.T, bID uuid.UUID, number string) uuid.UUID {
	t.Helper()
	uID := uuid.New()
	if err := e.exec(`INSERT INTO units (id, building_id, number, area_m2, status) VALUES (?, ?, ?, 100, 'active')`, uID, bID, number); err != nil {
		t.Fatalf("seed unit: %v", err)
	}
	if err := e.exec(`INSERT INTO unit_balances (unit_id, balance) VALUES (?, 0)`, uID); err != nil {
		// unit_balances may not be required for maintenance — ignore error if table missing in old schema
		_ = err
	}
	return uID
}

func (e *maintEnv) seedPerson(t *testing.T, bID uuid.UUID, phone, name string) uuid.UUID {
	t.Helper()
	pID := uuid.New()
	if err := e.exec(`INSERT INTO persons (id, building_id, full_name, phone) VALUES (?, ?, ?, ?)`, pID, bID, name, phone); err != nil {
		t.Fatalf("seed person: %v", err)
	}
	return pID
}

func (e *maintEnv) seedOccupancy(t *testing.T, unitID, personID uuid.UUID) uuid.UUID {
	t.Helper()
	oID := uuid.New()
	if err := e.exec(`INSERT INTO occupancies (id, unit_id, person_id, relationship, start_date) VALUES (?, ?, ?, 'tenant', CURRENT_DATE)`, oID, unitID, personID); err != nil {
		t.Fatalf("seed occupancy: %v", err)
	}
	return oID
}

func (e *maintEnv) seedPhoto(t *testing.T, uploader uuid.UUID) uuid.UUID {
	t.Helper()
	fID := uuid.New()
	if err := e.exec(`INSERT INTO files (id, path, content_type, size_bytes, uploaded_by) VALUES (?, ?, 'image/jpeg', 1234, ?)`, fID, "uploads/"+fID.String()+".jpg", uploader); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	return fID
}

func (e *maintEnv) do(t *testing.T, method, path, token string, body any) (int, map[string]any) {
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

func (e *maintEnv) get(t *testing.T, path, token string) (int, map[string]any) {
	t.Helper()
	return e.do(t, http.MethodGet, path, token, nil)
}
func (e *maintEnv) post(t *testing.T, path, token string, body any) (int, map[string]any) {
	t.Helper()
	return e.do(t, http.MethodPost, path, token, body)
}
func (e *maintEnv) patch(t *testing.T, path, token string, body any) (int, map[string]any) {
	t.Helper()
	return e.do(t, http.MethodPatch, path, token, body)
}

func maintNotifsFor(t *testing.T, e *maintEnv, userID uuid.UUID) int {
	t.Helper()
	var n int64
	if err := e.gormDB.Raw(`SELECT COUNT(*) FROM notifications WHERE user_id = ?`, userID).Scan(&n).Error; err != nil {
		t.Fatalf("notif count: %v", err)
	}
	return int(n)
}

func maintAuditsFor(t *testing.T, e *maintEnv, objectID uuid.UUID) []string {
	t.Helper()
	var acts []string
	if err := e.gormDB.Raw(`SELECT action FROM audit_logs WHERE object_id = ? ORDER BY created_at`, objectID).Scan(&acts).Error; err != nil {
		t.Fatalf("audit query: %v", err)
	}
	return acts
}

// --- submit + isolation + manager scope -----------------------------------

func TestMaintenanceSubmitAndIsolation(t *testing.T) {
	e := newMaintEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000001")
	bID := e.seedBuilding(t, mgrID)
	uID := e.seedUnit(t, bID, "1")

	resPhone := "09120000009"
	resID, resTok := e.seedUser(t, "resident", resPhone)
	pID := e.seedPerson(t, bID, resPhone, "رضا محمدی")
	e.seedOccupancy(t, uID, pID)

	// Another resident (different unit) for isolation checks.
	resPhone2 := "09120000010"
	resID2, resTok2 := e.seedUser(t, "resident", resPhone2)
	pID2 := e.seedPerson(t, bID, resPhone2, "علی حسینی")
	uID2 := e.seedUnit(t, bID, "2")
	e.seedOccupancy(t, uID2, pID2)
	_ = resID2

	// A manager without the building grant.
	_, otherMgrTok := e.seedUser(t, "manager", "09120000002")

	// Resident submits: minimal required fields → 201, status new.
	code, resp := e.post(t, "/api/v1/me/maintenance-requests", resTok, map[string]any{
		"title": "آسانسور لرزش دارد", "category": "elevator",
		"description": "لرزش هنگام حرکت", "location": "آسانسور اصلی",
		"priority": "urgent",
	})
	if code != http.StatusCreated {
		t.Fatalf("submit: %d %v", code, resp)
	}
	if resp["status"] != StatusNew || resp["priority"] != PriUrgent || resp["category"] != CatElevator {
		t.Fatalf("submit fields: %v", resp)
	}
	reqID := resp["id"].(string)

	// Resident own list: exactly one; other resident sees zero.
	if code, resp := e.get(t, "/api/v1/me/maintenance-requests", resTok); code != 200 || len(resp["items"].([]any)) != 1 {
		t.Fatalf("own list: %d %v", code, resp)
	}
	if code, resp := e.get(t, "/api/v1/me/maintenance-requests", resTok2); code != 200 || len(resp["items"].([]any)) != 0 {
		t.Fatalf("other resident list should be empty: %d %v", code, resp)
	}
	// Other resident cannot GET the request (403 isolation, not 404 leak).
	if code, _ := e.get(t, "/api/v1/me/maintenance-requests/"+reqID, resTok2); code != http.StatusForbidden {
		t.Fatalf("isolation breach: other resident GET = %d, want 403", code)
	}
	// Owner can GET own detail.
	if code, resp := e.get(t, "/api/v1/me/maintenance-requests/"+reqID, resTok); code != 200 || resp["id"] != reqID {
		t.Fatalf("own GET: %d %v", code, resp)
	}

	// Manager building list: sees the one request.
	if code, resp := e.get(t, "/api/v1/buildings/"+bID.String()+"/maintenance-requests", mgrTok); code != 200 || len(resp["items"].([]any)) != 1 {
		t.Fatalf("manager list: %d %v", code, resp)
	}
	// Foreign manager without grant → 403.
	if code, _ := e.get(t, "/api/v1/buildings/"+bID.String()+"/maintenance-requests", otherMgrTok); code != http.StatusForbidden {
		t.Fatalf("foreign manager list: %d, want 403", code)
	}
	// Resident cannot open manager list → 403.
	if code, _ := e.get(t, "/api/v1/buildings/"+bID.String()+"/maintenance-requests", resTok); code != http.StatusForbidden {
		t.Fatalf("resident manager list: %d, want 403", code)
	}

	// Submit validation: missing title → 400.
	if code, _ := e.post(t, "/api/v1/me/maintenance-requests", resTok, map[string]any{"title": "", "category": "elevator"}); code != http.StatusBadRequest {
		t.Fatalf("empty title: %d, want 400", code)
	}
	// Invalid category → 400.
	if code, _ := e.post(t, "/api/v1/me/maintenance-requests", resTok, map[string]any{"title": "تست", "category": "luxury"}); code != http.StatusBadRequest {
		t.Fatalf("bad category: %d, want 400", code)
	}
	// Photo binding: unknown file → 400.
	if code, _ := e.post(t, "/api/v1/me/maintenance-requests", resTok, map[string]any{"title": "با عکس", "category": "other", "photo_file_id": uuid.NewString()}); code != http.StatusBadRequest {
		t.Fatalf("unknown photo: %d, want 400", code)
	}
	// Photo binding: known file → 201 with stored path.
	photoID := e.seedPhoto(t, resID)
	if code, resp := e.post(t, "/api/v1/me/maintenance-requests", resTok, map[string]any{"title": "با عکس درست", "category": "other", "photo_file_id": photoID.String()}); code != http.StatusCreated || resp["photo_file"] == nil {
		t.Fatalf("photo submit: %d %v", code, resp)
	}
	// Unregistered resident (no occupancy) → 403.
	_, noOccTok := e.seedUser(t, "resident", "09120000011")
	if code, _ := e.post(t, "/api/v1/me/maintenance-requests", noOccTok, map[string]any{"title": "بدون سکونت", "category": "other"}); code != http.StatusForbidden {
		t.Fatalf("no-occupancy submit: %d, want 403", code)
	}
}

// --- state machine: legal transitions + 409 on illegal ----------------------

func TestMaintenanceStateMachine(t *testing.T) {
	e := newMaintEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000101")
	bID := e.seedBuilding(t, mgrID)
	uID := e.seedUnit(t, bID, "1")
	resPhone := "09120000109"
	resID, resTok := e.seedUser(t, "resident", resPhone)
	pID := e.seedPerson(t, bID, resPhone, "سارا")
	e.seedOccupancy(t, uID, pID)
	assigneePerson := e.seedPerson(t, bID, "09120000120", "تکنسین")

	// Helper: submit one request and return its id.
	submit := func(title string) string {
		code, resp := e.post(t, "/api/v1/me/maintenance-requests", resTok, map[string]any{"title": title, "category": "water"})
		if code != http.StatusCreated {
			t.Fatalf("submit %q: %d %v", title, code, resp)
		}
		return resp["id"].(string)
	}
	patchStatus := func(id, status string) (int, map[string]any) {
		return e.patch(t, "/api/v1/maintenance-requests/"+id, mgrTok, map[string]any{"status": status})
	}

	// Happy path: new → under_review → in_progress → done → closed.
	id := submit("مسیر کامل")
	for _, want := range []string{StatusUnderReview, StatusInProgress, StatusDone, StatusClosed} {
		code, resp := patchStatus(id, want)
		if code != 200 || resp["status"] != want {
			t.Fatalf("transition to %s: %d %v", want, code, resp)
		}
	}
	// closed is terminal: any further transition → 409.
	if code, _ := patchStatus(id, StatusInProgress); code != http.StatusConflict {
		t.Fatalf("closed terminal: %d, want 409", code)
	}

	// Illegal skip: new → in_progress (must go via under_review) → 409.
	id2 := submit("پرش غیرمجاز")
	if code, _ := patchStatus(id2, StatusInProgress); code != http.StatusConflict {
		t.Fatalf("skip new→in_progress: %d, want 409", code)
	}
	if code, _ := patchStatus(id2, StatusDone); code != http.StatusConflict {
		t.Fatalf("skip new→done: %d, want 409", code)
	}
	// Backwards: after moving to under_review, new→new is noop but under_review→new is 409.
	if code, _ := patchStatus(id2, StatusUnderReview); code != 200 {
		t.Fatalf("new→under_review: %d", code)
	}
	if code, _ := patchStatus(id2, StatusNew); code != http.StatusConflict {
		t.Fatalf("backwards under_review→new: %d, want 409", code)
	}
	// under_review → done is illegal (must go via in_progress) → 409.
	if code, _ := patchStatus(id2, StatusDone); code != http.StatusConflict {
		t.Fatalf("skip under_review→done: %d, want 409", code)
	}

	// Early close: under_review → closed allowed.
	id3 := submit("بستن زودهنگام از بررسی")
	if code, _ := patchStatus(id3, StatusUnderReview); code != 200 {
		t.Fatalf("to under_review: %d", code)
	}
	if code, resp := patchStatus(id3, StatusClosed); code != 200 || resp["status"] != StatusClosed {
		t.Fatalf("early close under_review→closed: %d %v", code, resp)
	}
	if respClosedAtMissing(t, e, id3) {
		t.Fatalf("closed_at must be set on close")
	}

	// Early close: in_progress → closed allowed.
	id4 := submit("بستن از در حال انجام")
	patchStatus(id4, StatusUnderReview)
	patchStatus(id4, StatusInProgress)
	if code, resp := patchStatus(id4, StatusClosed); code != 200 || resp["status"] != StatusClosed {
		t.Fatalf("early close in_progress→closed: %d %v", code, resp)
	}

	// Early close: done → closed allowed; done→in_progress backwards → 409.
	id5 := submit("برگشت از انجام‌شده")
	patchStatus(id5, StatusUnderReview)
	patchStatus(id5, StatusInProgress)
	patchStatus(id5, StatusDone)
	if code, _ := patchStatus(id5, StatusInProgress); code != http.StatusConflict {
		t.Fatalf("backwards done→in_progress: %d, want 409", code)
	}
	if code, resp := patchStatus(id5, StatusClosed); code != 200 || resp["status"] != StatusClosed {
		t.Fatalf("done→closed: %d %v", code, resp)
	}

	// Invalid status token → 400 (not 409 — 400 is validation before state machine).
	if code, _ := e.patch(t, "/api/v1/maintenance-requests/"+id2, mgrTok, map[string]any{"status": "flying"}); code != http.StatusBadRequest {
		t.Fatalf("invalid status: %d, want 400", code)
	}
	// Assignee / cost validation (400, not 409).
	if code, _ := e.patch(t, "/api/v1/maintenance-requests/"+id2, mgrTok, map[string]any{"assignee_person_id": uuid.NewString()}); code != http.StatusBadRequest {
		t.Fatalf("unknown assignee: %d, want 400", code)
	}
	if code, _ := e.patch(t, "/api/v1/maintenance-requests/"+id2, mgrTok, map[string]any{"recorded_cost": -5}); code != http.StatusBadRequest {
		t.Fatalf("negative cost: %d, want 400", code)
	}
	// Valid assignee + cost update (no status change) → 200.
	if code, resp := e.patch(t, "/api/v1/maintenance-requests/"+id2, mgrTok, map[string]any{"assignee_person_id": assigneePerson.String(), "recorded_cost": 500000, "notes": "یادداشت مدیر"}); code != 200 {
		t.Fatalf("assignee+cost: %d %v", code, resp)
	} else {
		if resp["assignee_person_id"] == nil || num(resp, "recorded_cost") != 500000 {
			t.Fatalf("assignee/cost not persisted: %v", resp)
		}
	}
	// Foreign manager cannot patch (403).
	_, foreignTok := e.seedUser(t, "manager", "09120000151")
	if code, _ := e.patch(t, "/api/v1/maintenance-requests/"+id2, foreignTok, map[string]any{"status": StatusInProgress}); code != http.StatusForbidden {
		t.Fatalf("foreign manager patch: %d, want 403", code)
	}
	// Resident cannot patch (403).
	if code, _ := e.patch(t, "/api/v1/maintenance-requests/"+id2, resTok, map[string]any{"status": StatusInProgress}); code != http.StatusForbidden {
		t.Fatalf("resident patch: %d, want 403", code)
	}
	_ = resID
}

func respClosedAtMissing(t *testing.T, e *maintEnv, id string) bool {
	t.Helper()
	var closedAt *time.Time
	if err := e.gormDB.Raw(`SELECT closed_at FROM maintenance_requests WHERE id = ?`, id).Scan(&closedAt).Error; err != nil {
		t.Fatalf("closed_at query: %v", err)
	}
	return closedAt == nil
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

// --- filters + notifications + audit ---------------------------------------

func TestMaintenanceFiltersNotificationsAndAudit(t *testing.T) {
	e := newMaintEnv(t)
	mgrID, mgrTok := e.seedUser(t, "manager", "09120000201")
	bID := e.seedBuilding(t, mgrID)
	uID := e.seedUnit(t, bID, "1")
	resPhone := "09120000209"
	resID, resTok := e.seedUser(t, "resident", resPhone)
	pID := e.seedPerson(t, bID, resPhone, "نیما")
	e.seedOccupancy(t, uID, pID)

	// Two requests with different categories/priorities.
	code, r1 := e.post(t, "/api/v1/me/maintenance-requests", resTok, map[string]any{"title": "آسانسور", "category": "elevator", "priority": "urgent"})
	if code != 201 {
		t.Fatalf("r1: %d %v", code, r1)
	}
	r1ID := r1["id"].(string)
	code, r2 := e.post(t, "/api/v1/me/maintenance-requests", resTok, map[string]any{"title": "لوله", "category": "water", "priority": "normal"})
	if code != 201 {
		t.Fatalf("r2: %d %v", code, r2)
	}
	_ = r2

	// Manager filters: category, priority.
	if code, resp := e.get(t, "/api/v1/buildings/"+bID.String()+"/maintenance-requests?category=elevator", mgrTok); code != 200 || len(resp["items"].([]any)) != 1 {
		t.Fatalf("filter category: %d %v", code, resp)
	}
	if code, resp := e.get(t, "/api/v1/buildings/"+bID.String()+"/maintenance-requests?priority=urgent", mgrTok); code != 200 || len(resp["items"].([]any)) != 1 {
		t.Fatalf("filter priority: %d %v", code, resp)
	}
	if code, resp := e.get(t, "/api/v1/buildings/"+bID.String()+"/maintenance-requests?status=new", mgrTok); code != 200 || len(resp["items"].([]any)) != 2 {
		t.Fatalf("filter status new: %d %v", code, resp)
	}
	// Change one status, then filter shows it.
	if code, _ := e.patch(t, "/api/v1/maintenance-requests/"+r1ID, mgrTok, map[string]any{"status": StatusUnderReview}); code != 200 {
		t.Fatalf("patch: %d", code)
	}
	if code, resp := e.get(t, "/api/v1/buildings/"+bID.String()+"/maintenance-requests?status=under_review", mgrTok); code != 200 || len(resp["items"].([]any)) != 1 {
		t.Fatalf("filter under_review: %d %v", code, resp)
	}

	// Notifications: one request_submitted per submit + one request_status_changed per transition, all to the submitter.
	beforeN := maintNotifsFor(t, e, resID)
	// Move r1 through two more transitions → two more notifications.
	e.patch(t, "/api/v1/maintenance-requests/"+r1ID, mgrTok, map[string]any{"status": StatusInProgress})
	e.patch(t, "/api/v1/maintenance-requests/"+r1ID, mgrTok, map[string]any{"status": StatusDone})
	afterN := maintNotifsFor(t, e, resID)
	if afterN-beforeN != 2 {
		t.Fatalf("notifications: before=%d after=%d, want +2 (one per transition)", beforeN, afterN)
	}
	var lastNotif struct {
		Type    string `gorm:"column:type"`
		RefType *string `gorm:"column:ref_type"`
		RefID   *string `gorm:"column:ref_id"`
	}
	if err := e.gormDB.Raw(`SELECT type, ref_type, ref_id FROM notifications WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`, resID).Scan(&lastNotif).Error; err != nil {
		t.Fatalf("last notif: %v", err)
	}
	if lastNotif.Type != "request_status_changed" {
		t.Fatalf("last notif type = %q, want request_status_changed", lastNotif.Type)
	}
	if lastNotif.RefType == nil || *lastNotif.RefType != "maintenance_request" {
		t.Fatalf("notif ref_type = %v, want maintenance_request", lastNotif.RefType)
	}

	// Audit: submit + each status change appends a row (submit=maintenance.create, moves=maintenance.status_changed).
	acts := maintAuditsFor(t, e, uuid.MustParse(r1ID))
	hasCreate := false
	statusChangedCount := 0
	for _, a := range acts {
		if a == "maintenance.create" {
			hasCreate = true
		}
		if a == "maintenance.status_changed" {
			statusChangedCount++
		}
	}
	if !hasCreate {
		t.Fatalf("audit missing maintenance.create: %v", acts)
	}
	if statusChangedCount < 3 {
		t.Fatalf("audit status_changed = %d, want ≥3 (under_review, in_progress, done): %v", statusChangedCount, acts)
	}
	_ = mgrID
}
