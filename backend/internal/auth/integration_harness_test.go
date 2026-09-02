package auth

import (
	"context"
	"database/sql"
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

	"hamsa/internal/platform/db"
	"hamsa/internal/platform/httpx"
)

// --- shared integration harness ---------------------------------------------
//
// Both US1 integration suites (auth_integration_test.go, authz_test.go) run
// against a throwaway PostgreSQL 16 with the real migrations applied, driving
// the real Gin handlers end-to-end. When Docker is unavailable (CI without
// docker.sock, developer laptop) the container start fails and the suite
// SKIPS instead of failing — the unit suite still guards the logic.

type loginAuditRecord struct {
	actorID    *uuid.UUID
	action     string
	objectType string
	objectID   *uuid.UUID
}

type fakeAuditor struct{ records []loginAuditRecord }

func (f *fakeAuditor) Append(_ context.Context, actorID *uuid.UUID, action, objectType string, objectID *uuid.UUID, _, _ any) error {
	f.records = append(f.records, loginAuditRecord{actorID: actorID, action: action, objectType: objectType, objectID: objectID})
	return nil
}

type authEnv struct {
	engine  *gin.Engine
	tokens  *TokenService
	users   *Repository
	auditor *fakeAuditor
	gormDB  *gorm.DB
}

// exec runs raw SQL against the test database (fixture seeding).
func (e *authEnv) exec(query string, args ...any) error {
	return e.gormDB.WithContext(context.Background()).Exec(query, args...).Error
}

// baseTestDSN is the fallback PostgreSQL used when no Docker engine is
// available: a reachable dev PostgreSQL (docker-compose credentials).
const baseTestDSN = "host=localhost port=5432 user=hamsa password=hamsa dbname=hamsa sslmode=disable"

func newAuthEnv(t *testing.T) *authEnv {
	t.Helper()

	dsn := os.Getenv("HAMSA_TEST_DSN")
	if dsn == "" {
		if !pgReachable(t) {
			t.Skip("skipping integration test — no Docker engine and no reachable PostgreSQL (set HAMSA_TEST_DSN)")
		}
		dsn = baseTestDSN
	}

	// Throwaway SCHEMA per test function so migrations, fixtures, and data
	// never touch shared objects — and no CREATEDB privilege is needed.
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

	// All connections (GORM + golang-migrate) resolve unqualified names in
	// the throwaway schema first.
	dsn = dsn + " search_path=" + schema
	if err := db.Migrate(dsn, "../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	gormDB, err := db.Connect(dsn, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	users := NewRepository(gormDB)
	tokens := NewTokenService([]byte("test-secret-32-bytes-long!!!!!!"),
		15*time.Minute, 30*24*time.Hour, &GormRefreshStore{DB: gormDB}, RealClock{})

	invites := NewInviteService(&GormInviteStore{DB: gormDB}, RealClock{})
	auditor := &fakeAuditor{}

	gin.SetMode(gin.TestMode)
	router := httpx.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), "dev")
	Register(router.Group("/api/v1/auth"), &Handler{
		Invites: invites,
		Tokens:  tokens,
		Users:   users,
		Scopes:  NewScopeResolver(gormDB),
		Auditor: auditor,
	})
	return &authEnv{engine: router, tokens: tokens, users: users, auditor: auditor, gormDB: gormDB}
}

// request performs an HTTP call against the engine. token may be empty.
func (e *authEnv) request(t *testing.T, method, path, body, token string) *httptest.ResponseRecorder {
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

// pgReachable probes the fallback PostgreSQL with a short timeout.
func pgReachable(t *testing.T) bool {
	t.Helper()
	dbh, err := sql.Open("pgx", baseTestDSN)
	if err != nil {
		return false
	}
	defer dbh.Close()
	dbh.SetConnMaxLifetime(time.Second)
	return dbh.Ping() == nil
}
