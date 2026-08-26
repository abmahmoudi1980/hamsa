package httpx

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter builds the Gin engine with the standard middleware stack
// (request-id, structured logging, recovery) and the liveness probe.
// Domain route groups (auth, building, billing, ...) are mounted onto the
// returned engine by their handlers as those are implemented.
func NewRouter(log *slog.Logger, env string) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(RequestID(), Logger(log), Recovery(log))

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
