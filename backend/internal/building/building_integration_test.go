package building

// US2 integration suite (T028): drives the real Gin handlers against a
// throwaway PostgreSQL schema with the real migrations applied — mirroring
// the auth package harness. Skips when neither Docker nor a reachable dev
// PostgreSQL exists (set HAMSA_TEST_DSN to force).

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

type bEnv struct {
	engine *gin.Engine
	gormDB *gorm.DB
	tokens *auth.TokenService
}

func (e *bEnv) exec(query string, args ...any) error {
	return e.gormDB.WithContext(context.Background()).Exec(query, args...).Error
}

const baseTestDSN = "host=localhost port=5432 user=hamsa password=hamsa dbname=hamsa sslmode=disable"

func newBuildingEnv(t *testing.T) *bEnv {
	t.Helper()

	dsn := os.Getenv("HAMSA_TEST_DSN")
	if dsn == "" {
		if !pgReachableB(t) {
			t.Skip("skipping integration test — no Docker engine and no reachable PostgreSQL (set HAMSA_TEST_DSN)")
		}
		dsn = baseTestDSN
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
	svc := NewService(NewRepository(gormDB))
	aud := audit.New(gormDB, log)

	gin.SetMode(gin.TestMode)
	router := httpx.NewRouter(log, "dev")
	authMW := auth.Authenticate(tokens, auth.NewRepository(gormDB))
	Register(router.Group("/api/v1", authMW, auth.RequireRole(auth.RoleManager)), svc, aud)
	return &bEnv{engine: router, gormDB: gormDB, tokens: tokens}
}

func pgReachableB(t *testing.T) bool {
	t.Helper()
	dbh, err := sql.Open("pgx", baseTestDSN)
	if err != nil {
		return false
	}
	defer dbh.Close()
	dbh.SetConnMaxLifetime(time.Second)
	return dbh.Ping() == nil
}

// seedUser inserts a user directly and returns an access token for it.
func (e *bEnv) seedUser(t *testing.T, role string) (uuid.UUID, string) {
	t.Helper()
	id := uuid.New()
	if err := e.exec(
		`INSERT INTO users (id, phone, role, name) VALUES (?, ?, ?, ?)`,
		id, "09"+fmt.Sprintf("%09d", id.ID()%1_000_000_000), role, "تست "+role,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	token, _, err := e.tokens.IssueAccessToken(id)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return id, token
}

func (e *bEnv) request(t *testing.T, method, path, body, token string) *httptest.ResponseRecorder {
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

func bodyJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode response %s: %v (%s)", rec.Body.String(), err, rec.Body.String())
	}
	return m
}

const unitPayload = `{"number":"%s","floor":%d,"area_m2":100,"status":"active","block":"A"}`

// TestUS2_BuildingLifecycle covers the quickstart Scenario 1 backbone:
// create → list scope → unit registry → duplicate 409 → edit audit trail.
func TestUS2_BuildingLifecycle(t *testing.T) {
	env := newBuildingEnv(t)

	mgrID, mgrTok := env.seedUser(t, auth.RoleManager)
	_, otherTok := env.seedUser(t, auth.RoleManager)
	_, resTok := env.seedUser(t, auth.RoleResident)

	// Resident tokens never reach manager routes.
	res := env.request(t, "GET", "/api/v1/buildings", "", resTok)
	if res.Code != http.StatusForbidden {
		t.Fatalf("resident on manager route: got %d want 403", res.Code)
	}

	// Create building → 201 and creator auto-grant in user_buildings.
	res = env.request(t, "POST", "/api/v1/buildings",
		`{"name":"ساختمان نگین","address":"تهران","floor_count":5}`, mgrTok)
	if res.Code != http.StatusCreated {
		t.Fatalf("create building: got %d want 201 (%s)", res.Code, res.Body.String())
	}
	b := bodyJSON(t, res)["id"].(string)
	bid, _ := uuid.Parse(b)

	var grants int64
	if err := env.gormDB.Raw(`SELECT count(*) FROM user_buildings WHERE user_id = ? AND building_id = ?`,
		mgrID, bid).Scan(&grants).Error; err != nil || grants != 1 {
		t.Fatalf("creator auto-grant: grants=%d err=%v", grants, err)
	}

	// Second manager: invisible list and 403 object access (existence-blind).
	res = env.request(t, "GET", "/api/v1/buildings", "", otherTok)
	if items := bodyJSON(t, res)["items"].([]any); len(items) != 0 {
		t.Fatalf("scope leak: other manager sees %d buildings", len(items))
	}
	res = env.request(t, "GET", "/api/v1/buildings/"+bid.String(), "", otherTok)
	if res.Code != http.StatusForbidden {
		t.Fatalf("cross-manager detail: got %d want 403", res.Code)
	}

	// 10 units created; duplicate number rejected with the Persian conflict.
	for i := 1; i <= 10; i++ {
		res = env.request(t, "POST", "/api/v1/buildings/"+bid.String()+"/units",
			fmt.Sprintf(unitPayload, fmt.Sprintf("%d", i), i), mgrTok)
		if res.Code != http.StatusCreated {
			t.Fatalf("create unit %d: got %d (%s)", i, res.Code, res.Body.String())
		}
	}
	dup := env.request(t, "POST", "/api/v1/buildings/"+bid.String()+"/units",
		fmt.Sprintf(unitPayload, "1", 11), mgrTok)
	if dup.Code != http.StatusConflict {
		t.Fatalf("duplicate unit: got %d want 409", dup.Code)
	}
	errObj := bodyJSON(t, dup)["error"].(map[string]any)
	if msg, _ := errObj["message"].(string); !strings.Contains(msg, "تکراری") {
		t.Fatalf("duplicate message not Persian-conflict: %q", msg)
	}

	// Edit a unit → audited; history endpoint returns both create + update.
	unitListRes := env.request(t, "GET", "/api/v1/buildings/"+bid.String()+"/units", "", mgrTok)
	items := bodyJSON(t, unitListRes)["items"].([]any)
	if len(items) != 10 {
		t.Fatalf("unit count: got %d want 10", len(items))
	}
	first := items[0].(map[string]any)["id"].(string)
	upd := env.request(t, "PUT", "/api/v1/units/"+first,
		`{"number":"1","area_m2":120,"status":"occupied","floor":1}`, mgrTok)
	if upd.Code != http.StatusOK {
		t.Fatalf("update unit: got %d (%s)", upd.Code, upd.Body.String())
	}

	hist := env.request(t, "GET", "/api/v1/units/"+first+"/history", "", mgrTok)
	if hist.Code != http.StatusOK {
		t.Fatalf("history: got %d", hist.Code)
	}
	hitems := bodyJSON(t, hist)["items"].([]any)
	actions := map[string]bool{}
	for _, it := range hitems {
		actions[it.(map[string]any)["action"].(string)] = true
	}
	if !actions["unit.create"] || !actions["unit.update"] {
		t.Fatalf("audit trail incomplete: %v", actions)
	}

	// Search & filter.
	q := env.request(t, "GET", "/api/v1/buildings/"+bid.String()+"/units?q=7", "", mgrTok)
	if n := len(bodyJSON(t, q)["items"].([]any)); n != 1 {
		t.Fatalf("search q=7: got %d want 1", n)
	}
	fl := env.request(t, "GET", "/api/v1/buildings/"+bid.String()+"/units?floor=3&block=A", "", mgrTok)
	if n := len(bodyJSON(t, fl)["items"].([]any)); n != 1 {
		t.Fatalf("filter floor/block: got %d want 1", n)
	}
	st := env.request(t, "GET", "/api/v1/buildings/"+bid.String()+"/units?status=active", "", mgrTok)
	if n := len(bodyJSON(t, st)["items"].([]any)); n != 9 { // unit 1 flipped to occupied
		t.Fatalf("filter status=active: got %d want 9", n)
	}
	del := env.request(t, "DELETE", "/api/v1/units/"+first, "", mgrTok)
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete unit: got %d", del.Code)
	}
	reuse := env.request(t, "POST", "/api/v1/buildings/"+bid.String()+"/units",
		fmt.Sprintf(unitPayload, "1", 1), mgrTok)
	if reuse.Code != http.StatusCreated {
		t.Fatalf("reuse deleted number: got %d (%s)", reuse.Code, reuse.Body.String())
	}
	listAfter := env.request(t, "GET", "/api/v1/buildings/"+bid.String()+"/units", "", mgrTok)
	if n := int(bodyJSON(t, listAfter)["total"].(float64)); n != 10 {
		t.Fatalf("live units after archive+recreate: got %d want 10", n)
	}
}

// TestUS2_ValidationErrors covers area > 0, invalid status, missing name.
func TestUS2_ValidationErrors(t *testing.T) {
	env := newBuildingEnv(t)
	_, tok := env.seedUser(t, auth.RoleManager)

	res := env.request(t, "POST", "/api/v1/buildings", `{"name":""}`, tok)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("empty name: got %d want 400", res.Code)
	}

	res = env.request(t, "POST", "/api/v1/buildings", `{"name":"ساختمان"}`, tok)
	bid := bodyJSON(t, res)["id"].(string)

	area := env.request(t, "POST", "/api/v1/buildings/"+bid+"/units",
		`{"number":"1","area_m2":0,"floor":1}`, tok)
	if area.Code != http.StatusBadRequest {
		t.Fatalf("zero area: got %d want 400", area.Code)
	}
	status := env.request(t, "POST", "/api/v1/buildings/"+bid+"/units",
		`{"number":"2","area_m2":50,"floor":1,"status":"burned"}`, tok)
	if status.Code != http.StatusBadRequest {
		t.Fatalf("bad status: got %d want 400", status.Code)
	}
}

// TestUS2_Unauthenticated requires a bearer token everywhere.
func TestUS2_Unauthenticated(t *testing.T) {
	env := newBuildingEnv(t)
	for _, tc := range [][2]string{
		{"GET", "/api/v1/buildings"},
		{"POST", "/api/v1/buildings"},
	} {
		res := env.request(t, tc[0], tc[1], "{}", "")
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without token: got %d want 401", tc[0], tc[1], res.Code)
		}
	}
}

var _ = context.Background // keep imports aligned if harness evolves
