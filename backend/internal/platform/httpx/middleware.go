package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDHeader is the header carrying the per-request correlation id.
const RequestIDHeader = "X-Request-ID"

// ctxRequestID is the gin context key for the request id.
const ctxRequestID = "request_id"

// RequestID assigns (or honors an inbound) request id and sets the response
// header so clients and logs can correlate.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(ctxRequestID, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// Logger logs one structured line per request (method, path, status,
// latency, request id, client ip).
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		attrs := []any{
			slog.String("request_id", RequestIDOf(c)),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", query),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", latency),
			slog.String("client_ip", c.ClientIP()),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("errors", c.Errors.String()))
		}
		switch {
		case c.Writer.Status() >= 500:
			log.Error("http request", attrs...)
		case c.Writer.Status() >= 400:
			log.Warn("http request", attrs...)
		default:
			log.Info("http request", attrs...)
		}
	}
}

// Recovery converts panics into the JSON error envelope instead of a naked
// 500 with a stack trace.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic recovered",
					slog.String("request_id", RequestIDOf(c)),
					slog.Any("panic", rec),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": ErrInternal,
				})
			}
		}()
		c.Next()
	}
}

// RequestIDOf returns the request id stored in the gin context.
func RequestIDOf(c *gin.Context) string {
	if v, ok := c.Get(ctxRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
