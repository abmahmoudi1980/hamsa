package httpx

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RouterOptions carries the runtime configuration NewRouter needs. It is
// passed as a single struct (rather than more positional args) because the
// test harnesses construct the engine directly and would otherwise have to
// thread production-only settings through.
type RouterOptions struct {
	// AllowedOrigins is the CORS allowlist (config.cors.allowed_origins).
	// Empty means no CORS headers — the production posture, since the web
	// client is served same-origin with the API.
	AllowedOrigins []string
}

// NewRouter builds the Gin engine with the standard middleware stack
// (request-id, structured logging, recovery, CORS) and the liveness probe.
// Domain route groups (auth, building, billing, ...) are mounted onto the
// returned engine by their handlers as those are implemented.
//
// Callers in tests pass the zero value, which yields no CORS headers.
func NewRouter(log *slog.Logger, env string, opts RouterOptions) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(RequestID(), Logger(log), Recovery(log), CORS(opts.AllowedOrigins))

	// Liveness probe — outside /api/v1; must not depend on the database so
	// it reports process health even during a transient DB blip.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Unmatched routes still honor the JSON error envelope (contracts/api.md:
	// every non-2xx is {error:{code,message,details}}).
	r.NoRoute(func(c *gin.Context) {
		WriteError(c, NotFound("مسیر مورد نظر یافت نشد."))
	})

	return r
}
