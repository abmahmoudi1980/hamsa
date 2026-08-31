package announcement

// T075 — US8 integration suite: drives the real Gin handlers against a
// throwaway PostgreSQL schema with the real migrations applied (same harness
// pattern as maintenance/expense/billing). Covers contracts/api.md
// "Announcements" and quickstart Scenario 6 step 3:
//
//   - audience targeting (all/block/floor/unit) — block-A announcement
//     visible only to block-A residents; floor-2 only to floor 2, etc.
//   - publish/expire window visibility (before publish, after expire hidden)
//   - read-tracking / unread counts (POST /announcements/{id}/read, idempotent)
//   - manager scope: non-granted manager gets 403; resident cannot access manager routes
//   - notifications: announcement_published emitted to targeted residents
//   - audit entries for publish/update/delete

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

type annEnv struct {
	engine *gin.Engine
	gormDB *gorm.DB
	tokens *auth.TokenService
	svc    *Service
}

func (e *annEnv) exec(query string, args ...any) error {
	return e.gormDB.WithContext(context.Background()).Exec(query, args...).Error
}

const annBaseDSN = "host=localhost port=5432 user=hamsa password=hamsa dbname=hamsa sslmode=disable"

func newAnnEnv(t *testing.T) *annEnv {
	t.Helper()

	dsn := os.Getenv("HAMSA_TEST_DSN")
	if dsn == "" {
		if !pgReachableAnn(t) {
			t.Skip("skipping integration test — no Docker engine and no reachable PostgreSQL (set HAMSA_TEST_DSN)")
		}
		dsn = annBaseDSN
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
	repo := NewRepository(gormDB)
	svc := NewService(repo, notifSvc, audSvc)

	gin.SetMode(gin.TestMode)
	router := httpx.NewRouter(log, "dev")
	authMW := auth.Authenticate(tokens, auth.NewRepository(gormDB))
	authed := router.Group("/api/v1", authMW)
	Register(authed, svc)

	return &annEnv{engine: router, gormDB: gormDB, tokens: tokens, svc: svc}
}

func pgReachableAnn(t *testing.T) bool {
	t.Helper()
	dbh, err := sql.Open("pgx", annBaseDSN)
	if err != nil {
		return false
	}
	defer dbh.Close()
	dbh.SetConnMaxLifetime(time.Second)
	return dbh.Ping() == nil
}

func (e *annEnv) seedUser(t *testing.T, role, phone string) (uuid.UUID, string) {
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

func (e *annEnv) seedBuilding(t *testing.T, mgrID uuid.UUID) uuid.UUID {
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

func (e *annEnv) seedUnit(t *testing.T, bID uuid.UUID, number, block string, floor int) uuid.UUID {
	t.Helper()
	uID := uuid.New()
	var blockVal any
	if block != "" {
		blockVal = block
	}
	if err := e.exec(`INSERT INTO units (id, building_id, number, block, floor, area_m2, status) VALUES (?, ?, ?, ?, ?, 80, 'occupied')`, uID, bID, number, blockVal, floor); err != nil {
		t.Fatalf("seedUnit %s: %v", number, err)
	}
	return uID
}

func (e *annEnv) seedPerson(t *testing.T, bID uuid.UUID, phone, name string) uuid.UUID {
	t.Helper()
	pID := uuid.New()
	if err := e.exec(`INSERT INTO persons (id, building_id, full_name, phone) VALUES (?, ?, ?, ?)`, pID, bID, name, phone); err != nil {
		t.Fatalf("seedPerson: %v", err)
	}
	return pID
}

func (e *annEnv) seedOccupancy(t *testing.T, unitID, personID uuid.UUID) {
	t.Helper()
	oID := uuid.New()
	if err := e.exec(`INSERT INTO occupancies (id, unit_id, person_id, relationship, start_date) VALUES (?, ?, ?, 'tenant', CURRENT_DATE)`, oID, unitID, personID); err != nil {
		t.Fatalf("seedOccupancy: %v", err)
	}
}

func (e *annEnv) do(t *testing.T, method, path, token string, body any) (int, map[string]any) {
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
			// Try array envelope fallback: keep raw for debug, but out stays nil
			out = map[string]any{"_raw": w.Body.String()}
		}
	} else {
		out = map[string]any{}
	}
	return w.Code, out
}

func (e *annEnv) post(t *testing.T, path, token string, body any) (int, map[string]any) {
	return e.do(t, http.MethodPost, path, token, body)
}
func (e *annEnv) get(t *testing.T, path, token string) (int, map[string]any) {
	return e.do(t, http.MethodGet, path, token, nil)
}
func (e *annEnv) put(t *testing.T, path, token string, body any) (int, map[string]any) {
	return e.do(t, http.MethodPut, path, token, body)
}
func (e *annEnv) del(t *testing.T, path, token string) (int, map[string]any) {
	return e.do(t, http.MethodDelete, path, token, nil)
}

func annNotifsFor(t *testing.T, e *annEnv, userID uuid.UUID) int {
	t.Helper()
	var n int64
	if err := e.gormDB.WithContext(context.Background()).Table("notifications").
		Where("user_id = ? AND type = 'announcement_published'", userID).Count(&n).Error; err != nil {
		t.Fatalf("count notifs: %v", err)
	}
	return int(n)
}

func annAuditsFor(t *testing.T, e *annEnv, objectID uuid.UUID) []string {
	t.Helper()
	var rows []struct {
		Action string `gorm:"column:action"`
	}
	if err := e.gormDB.WithContext(context.Background()).Table("audit_logs").
		Where("object_type = 'announcement' AND object_id = ?", objectID).Order("created_at ASC").Find(&rows).Error; err != nil {
		t.Fatalf("audit query: %v", err)
	}
	acts := make([]string, len(rows))
	for i, r := range rows {
		acts[i] = r.Action
	}
	return acts
}

func strPtr(s string) *string { return &s }

// --- audience targeting ------------------------------------------------------

func TestAnnouncementAudienceTargeting(t *testing.T) {
	e := newAnnEnv(t)

	mgrID, mgrTok := e.seedUser(t, "manager", "09120000001")
	bID := e.seedBuilding(t, mgrID)

	// Two blocks, two floors, four units.
	uA1 := e.seedUnit(t, bID, "1", "A", 1)
	uA2 := e.seedUnit(t, bID, "2", "A", 2)
	uB1 := e.seedUnit(t, bID, "3", "B", 1)
	_ = e.seedUnit(t, bID, "4", "B", 2)

	// Residents: R-A on unit A1 (block A, floor 1), R-B on B1 (block B, floor1), R-F2 on A2 (block A, floor2)
	_, resATok := e.seedUser(t, "resident", "09120000002")
	pA := e.seedPerson(t, bID, "09120000002", "ساکن A")
	e.seedOccupancy(t, uA1, pA)

	resBID, resBTok := e.seedUser(t, "resident", "09120000003")
	pB := e.seedPerson(t, bID, "09120000003", "ساکن B")
	e.seedOccupancy(t, uB1, pB)

	_, resF2Tok := e.seedUser(t, "resident", "09120000004")
	pF2 := e.seedPerson(t, bID, "09120000004", "ساکن F2")
	e.seedOccupancy(t, uA2, pF2)

	// Manager publishes 4 announcements.
	toCreate := []struct {
		title string
		typ   string
		val   *string
	}{
		{"همه", "all", nil},
		{"بلوک A", "block", strPtr("A")},
		{"طبقه 2", "floor", strPtr("2")},
		{"واحد 3 (B1)", "unit", func() *string { s := uB1.String(); return &s }()},
	}
	for _, tc := range toCreate {
		body := map[string]any{"title": tc.title, "body": "متن اطلاعیه", "audience_type": tc.typ}
		if tc.val != nil {
			body["audience_value"] = *tc.val
		}
		code, out := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID), mgrTok, body)
		if code != http.StatusCreated {
			t.Fatalf("create %q: got %d want 201: out=%v", tc.title, code, out)
		}
	}

	// Resident A (block A floor1) → all + block A = 2
	code, out := e.get(t, "/api/v1/me/announcements", resATok)
	if code != http.StatusOK {
		t.Fatalf("resA list: got %d want 200: %v", code, out)
	}
	if total := int(out["total"].(float64)); total != 2 {
		t.Fatalf("resA total=%d want 2: out=%v", total, out)
	}

	// Resident B (block B floor1 + unit 3 specific) → all + block?? block B not announced as block, but unit-specific matches B1, so all + unit =2
	// Actually we created unit targeting uB1, so R-B should see all + unit =2
	code, out = e.get(t, "/api/v1/me/announcements", resBTok)
	if code != http.StatusOK {
		t.Fatalf("resB list: got %d want 200: %v", code, out)
	}
	if total := int(out["total"].(float64)); total != 2 {
		t.Fatalf("resB total=%d want 2 (all+unit): out=%v", total, out)
	}

	// Resident F2 (block A floor2) → all + block A + floor2 =3
	code, out = e.get(t, "/api/v1/me/announcements", resF2Tok)
	if code != http.StatusOK {
		t.Fatalf("resF2 list: got %d want 200: %v", code, out)
	}
	if total := int(out["total"].(float64)); total != 3 {
		t.Fatalf("resF2 total=%d want 3 (all+blockA+floor2): out=%v", total, out)
	}

	// Notification: block-A announcement should have reached R-A and R-F2 but not R-B
	if n := annNotifsFor(t, e, resBID); n != 2 {
		// resB got all (1) + unit (1) =2 ; sanity check that targeting produced notifications
		t.Fatalf("notif count for R-B=%d want 2", n)
	}
}

// --- publish / expire window ------------------------------------------------

func TestAnnouncementPublishExpireWindow(t *testing.T) {
	e := newAnnEnv(t)

	mgrID, mgrTok := e.seedUser(t, "manager", "09120000011")
	bID := e.seedBuilding(t, mgrID)
	u1 := e.seedUnit(t, bID, "1", "A", 1)
	resID, resTok := e.seedUser(t, "resident", "09120000012")
	p := e.seedPerson(t, bID, "09120000012", "ساکن")
	e.seedOccupancy(t, u1, p)

	now := time.Now().UTC()
	past := now.Add(-2 * time.Hour).Format(time.RFC3339)
	future := now.Add(2 * time.Hour).Format(time.RFC3339)
	farFuture := now.Add(4 * time.Hour).Format(time.RFC3339)
	farPast := now.Add(-4 * time.Hour).Format(time.RFC3339)

	// 1) Not yet published (future publish_at) → hidden
	{
		body := map[string]any{"title": "آینده", "body": "متن", "audience_type": "all", "publish_at": future, "expire_at": farFuture}
		code, _ := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID), mgrTok, body)
		if code != http.StatusCreated {
			t.Fatalf("create future-publish: got %d want 201", code)
		}
	}
	// 2) Already expired → hidden
	{
		body := map[string]any{"title": "منقضی", "body": "متن", "audience_type": "all", "publish_at": farPast, "expire_at": past}
		code, _ := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID), mgrTok, body)
		if code != http.StatusCreated {
			t.Fatalf("create expired: got %d want 201", code)
		}
	}
	// 3) Visible window → should appear
	{
		body := map[string]any{"title": "فعال", "body": "متن", "audience_type": "all", "publish_at": past, "expire_at": farFuture}
		code, _ := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID), mgrTok, body)
		if code != http.StatusCreated {
			t.Fatalf("create visible: got %d want 201", code)
		}
	}
	// 4) No window (always visible)
	{
		body := map[string]any{"title": "همیشگی", "body": "متن", "audience_type": "all"}
		code, _ := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID), mgrTok, body)
		if code != http.StatusCreated {
			t.Fatalf("create always: got %d want 201", code)
		}
	}

	code, out := e.get(t, "/api/v1/me/announcements", resTok)
	if code != http.StatusOK {
		t.Fatalf("resident list window: got %d want 200: %v", code, out)
	}
	if total := int(out["total"].(float64)); total != 2 {
		t.Fatalf("window total=%d want 2 (active+always): %v", total, out)
	}
	// Verify titles are the visible ones
	items := out["items"].([]any)
	titles := map[string]bool{}
	for _, it := range items {
		m := it.(map[string]any)
		titles[m["title"].(string)] = true
	}
	if !titles["فعال"] || !titles["همیشگی"] {
		t.Fatalf("visible titles mismatch: %v", titles)
	}
	if titles["آینده"] || titles["منقضی"] {
		t.Fatalf("hidden announcements leaked: %v", titles)
	}

	// Notification guard: only in-window creations notify. آینده (future
	// publish_at) and منقضی (already expired) must NOT notify; فعال and
	// همیشگی must.
	if n := annNotifsFor(t, e, resID); n != 2 {
		t.Fatalf("notifications after window creates=%d want 2 (active+always only)", n)
	}
}

// --- read tracking -----------------------------------------------------------

func TestAnnouncementReadTracking(t *testing.T) {
	e := newAnnEnv(t)

	mgrID, mgrTok := e.seedUser(t, "manager", "09120000021")
	bID := e.seedBuilding(t, mgrID)
	u1 := e.seedUnit(t, bID, "1", "A", 1)
	resID, resTok := e.seedUser(t, "resident", "09120000022")
	p := e.seedPerson(t, bID, "09120000022", "ساکن")
	e.seedOccupancy(t, u1, p)

	// Publish one announcement
	var annID string
	{
		body := map[string]any{"title": "قابل خواندن", "body": "متن", "audience_type": "all"}
		code, out := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID), mgrTok, body)
		if code != http.StatusCreated {
			t.Fatalf("create: got %d want 201: %v", code, out)
		}
		annID = out["id"].(string)
	}

	// Initially unread
	code, out := e.get(t, "/api/v1/me/announcements", resTok)
	if code != http.StatusOK {
		t.Fatalf("list before read: %d %v", code, out)
	}
	items := out["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items len=%d want 1", len(items))
	}
	if isRead, _ := items[0].(map[string]any)["is_read"].(bool); isRead {
		t.Fatalf("initial is_read should be false")
	}

	// Mark read (contracts: POST /announcements/{id}/read)
	code, _ = e.post(t, fmt.Sprintf("/api/v1/announcements/%s/read", annID), resTok, nil)
	if code != http.StatusNoContent {
		t.Fatalf("mark read: got %d want 204", code)
	}

	// Now read
	code, out = e.get(t, "/api/v1/me/announcements", resTok)
	if code != http.StatusOK {
		t.Fatalf("list after read: %d %v", code, out)
	}
	items = out["items"].([]any)
	if isRead, _ := items[0].(map[string]any)["is_read"].(bool); !isRead {
		t.Fatalf("after mark is_read should be true")
	}

	// Idempotent second mark
	code, _ = e.post(t, fmt.Sprintf("/api/v1/announcements/%s/read", annID), resTok, map[string]any{})
	if code != http.StatusNoContent {
		t.Fatalf("second mark read: got %d want 204", code)
	}

	// Unread count via repo helper (indirect via list with unread filter not implemented; verify via DB)
	var cnt int64
	if err := e.gormDB.WithContext(context.Background()).Table("announcement_reads").Where("user_id = ?", resID).Count(&cnt).Error; err != nil {
		t.Fatalf("count reads: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("reads cnt=%d want 1", cnt)
	}

	// Other resident cannot mark an announcement not visible to them
	bID2 := e.seedBuilding(t, mgrID) // manager has second building, but this resident is not in it
	// Actually manager already has grant to bID2, but resident not occupant there
	uOther := e.seedUnit(t, bID2, "10", "A", 1)
	_ = uOther
	var ann2ID string
	{
		body := map[string]any{"title": "ساختمان دیگر", "body": "متن", "audience_type": "all"}
		code, out := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID2), mgrTok, body)
		if code != http.StatusCreated {
			t.Fatalf("create second building ann: %d %v", code, out)
		}
		ann2ID = out["id"].(string)
	}
	code, _ = e.post(t, fmt.Sprintf("/api/v1/announcements/%s/read", ann2ID), resTok, nil)
	if code != http.StatusNotFound {
		t.Fatalf("mark unreadable announcement: got %d want 404", code)
	}
}

// --- unread count beyond page size -------------------------------------------

func TestAnnouncementUnreadCountBeyondPageSize(t *testing.T) {
	e := newAnnEnv(t)

	mgrID, mgrTok := e.seedUser(t, "manager", "09120000041")
	bID := e.seedBuilding(t, mgrID)
	u1 := e.seedUnit(t, bID, "1", "A", 1)
	resID, resTok := e.seedUser(t, "resident", "09120000042")
	p := e.seedPerson(t, bID, "09120000042", "ساکن")
	e.seedOccupancy(t, u1, p)

	// 25 always-visible announcements (resident page size is 20).
	for i := range 25 {
		body := map[string]any{"title": fmt.Sprintf("اطلاعیه %d", i+1), "body": "متن", "audience_type": "all"}
		if code, out := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID), mgrTok, body); code != http.StatusCreated {
			t.Fatalf("create %d: %d %v", i+1, code, out)
		}
	}

	res := &auth.User{ID: resID, Phone: "09120000042"}
	unread, err := e.svc.UnreadCount(context.Background(), res)
	if err != nil {
		t.Fatalf("unread count: %v", err)
	}
	if unread != 25 {
		t.Fatalf("unread=%d want 25 (must not be capped by the page size)", unread)
	}

	// Read the first page (20 items) → unread must drop to exactly 5.
	code, out := e.get(t, "/api/v1/me/announcements", resTok)
	if code != http.StatusOK {
		t.Fatalf("list: %d %v", code, out)
	}
	for _, it := range out["items"].([]any) {
		id := it.(map[string]any)["id"].(string)
		if code, _ := e.post(t, "/api/v1/announcements/"+id+"/read", resTok, nil); code != http.StatusNoContent {
			t.Fatalf("mark read: %d", code)
		}
	}
	unread, err = e.svc.UnreadCount(context.Background(), res)
	if err != nil {
		t.Fatalf("unread count after reads: %v", err)
	}
	if unread != 5 {
		t.Fatalf("unread after reading first page=%d want 5", unread)
	}
}

// --- manager CRUD, notifications, audit -------------------------------------

func TestAnnouncementManagerCRUDAndAudit(t *testing.T) {
	e := newAnnEnv(t)

	mgrID, mgrTok := e.seedUser(t, "manager", "09120000031")
	bID := e.seedBuilding(t, mgrID)
	u1 := e.seedUnit(t, bID, "1", "A", 1)

	resID, _ := e.seedUser(t, "resident", "09120000032")
	p := e.seedPerson(t, bID, "09120000032", "ساکن")
	e.seedOccupancy(t, u1, p)

	// Non-granted manager → 403
	_, otherMgrTok := e.seedUser(t, "manager", "09120000033")
	// otherMgr has no building grant
	{
		body := map[string]any{"title": "x", "body": "y", "audience_type": "all"}
		code, _ := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID), otherMgrTok, body)
		if code != http.StatusForbidden {
			t.Fatalf("non-granted create: got %d want 403", code)
		}
	}

	// Create
	var annID string
	{
		body := map[string]any{"title": "اطلاعیه اول", "body": "متن اول", "audience_type": "all"}
		code, out := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID), mgrTok, body)
		if code != http.StatusCreated {
			t.Fatalf("create: %d %v", code, out)
		}
		annID = out["id"].(string)
		aid, _ := uuid.Parse(annID)
		if acts := annAuditsFor(t, e, aid); len(acts) == 0 || acts[0] != "announcement.publish" {
			t.Fatalf("audit after publish: %v", acts)
		}
		if n := annNotifsFor(t, e, resID); n != 1 {
			t.Fatalf("notification after publish: got %d want 1", n)
		}
	}

	// Update
	{
		body := map[string]any{"title": "ویرایش شده", "body": "متن جدید", "audience_type": "all"}
		code, out := e.put(t, fmt.Sprintf("/api/v1/announcements/%s", annID), mgrTok, body)
		if code != http.StatusOK {
			t.Fatalf("update: %d %v", code, out)
		}
		if out["title"].(string) != "ویرایش شده" {
			t.Fatalf("title after update: %v", out["title"])
		}
		aid, _ := uuid.Parse(annID)
		acts := annAuditsFor(t, e, aid)
		found := false
		for _, a := range acts {
			if a == "announcement.update" {
				found = true
			}
		}
		if !found {
			t.Fatalf("audit after update missing: %v", acts)
		}
	}

	// Resident cannot hit manager PUT → 403 (auth middleware)
	_, resTok2 := e.seedUser(t, "resident", "09120000034")
	p2 := e.seedPerson(t, bID, "09120000034", "ساکن2")
	u2 := e.seedUnit(t, bID, "2", "A", 1)
	e.seedOccupancy(t, u2, p2)
	{
		body := map[string]any{"title": "تلاش ساکن", "body": "x", "audience_type": "all"}
		code, _ := e.put(t, fmt.Sprintf("/api/v1/announcements/%s", annID), resTok2, body)
		if code != http.StatusForbidden {
			t.Fatalf("resident update: got %d want 403", code)
		}
	}

	// Delete
	{
		code, _ := e.del(t, fmt.Sprintf("/api/v1/announcements/%s", annID), mgrTok)
		if code != http.StatusNoContent {
			t.Fatalf("delete: got %d want 204", code)
		}
		aid, _ := uuid.Parse(annID)
		acts := annAuditsFor(t, e, aid)
		found := false
		for _, a := range acts {
			if a == "announcement.delete" {
				found = true
			}
		}
		if !found {
			t.Fatalf("audit after delete missing: %v", acts)
		}
		// Get after delete → 404
		code, _ = e.get(t, fmt.Sprintf("/api/v1/announcements/%s", annID), mgrTok)
		if code != http.StatusNotFound {
			t.Fatalf("get after delete: got %d want 404", code)
		}
	}

	// Validation: all with value → 400
	{
		body := map[string]any{"title": "بد", "body": "متن", "audience_type": "all", "audience_value": "A"}
		code, _ := e.post(t, fmt.Sprintf("/api/v1/buildings/%s/announcements", bID), mgrTok, body)
		if code != http.StatusBadRequest {
			t.Fatalf("all with value: got %d want 400", code)
		}
	}

	_ = mgrID
}
