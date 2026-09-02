package audit

// Best-effort audit guarantee (002 spec FR-020 / US4 scenario 4): when the
// audit store fails, audited operations still succeed — failures are logged
// and dropped, never surfaced to the client.

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	postgres "gorm.io/driver/postgres"
)

// deadDB returns a *gorm.DB pointing at a refused port: sql.Open is lazy, so
// gorm.Open succeeds and every query fails fast at connect time (localhost
// connection refused, ~ms).
func deadDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=127.0.0.1 port=1 user=hamsa password=hamsa dbname=hamsa sslmode=disable connect_timeout=1",
	}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}
	return gdb
}

func TestMiddlewareSurvivesAuditFailure(t *testing.T) {
	svc := New(deadDB(t), slog.New(slog.NewTextHandler(io.Discard, nil)))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	objectID := uuid.New()
	r.POST("/thing", svc.Middleware("thing.create", "thing"), func(c *gin.Context) {
		c.Set(CtxObjectID, objectID)
		c.Set(CtxAfter, map[string]any{"id": objectID})
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/thing", nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("audit failure leaked to client: got %d want 201 body=%s", rec.Code, rec.Body)
	}
}

func TestAppendReturnsErrorOnDeadStore(t *testing.T) {
	// Append itself reports the error to its caller — the invite handler
	// ignores it deliberately (best-effort), while Middleware logs it.
	svc := New(deadDB(t), slog.New(slog.NewTextHandler(io.Discard, nil)))
	actor := uuid.New()
	if err := svc.Append(t.Context(), &actor, "invite.issued", "invite", nil, nil,
		map[string]any{"phone": "09120000000"}); err == nil {
		t.Fatal("Append on dead store: want error, got nil")
	}
}
