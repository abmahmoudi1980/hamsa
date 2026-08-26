// Package audit provides the append-only audit trail (FR-038).
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/auth"
)

// Service appends append-only audit entries (migration 0001 audit_logs).
type Service struct {
	db  *gorm.DB
	log *slog.Logger
}

// New returns an audit service. A nil log defaults to slog.Default.
func New(db *gorm.DB, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{db: db, log: log}
}

// Append records an audit entry. actorID may be nil for system actions;
// before/after are JSON-marshalled (nil → NULL). objectType/objectID identify
// the affected object (e.g. "unit", "invoice").
func (s *Service) Append(ctx context.Context, actorID *uuid.UUID, action, objectType string, objectID *uuid.UUID, before, after any) error {
	var b, a any
	if before != nil {
		raw, err := json.Marshal(before)
		if err != nil {
			return fmt.Errorf("marshal before: %w", err)
		}
		b = string(raw)
	}
	if after != nil {
		raw, err := json.Marshal(after)
		if err != nil {
			return fmt.Errorf("marshal after: %w", err)
		}
		a = string(raw)
	}

	return s.db.WithContext(ctx).Exec(
		`INSERT INTO audit_logs (user_id, action, object_type, object_id, before_value, after_value)
		 VALUES (?, ?, ?, ?, ?::jsonb, ?::jsonb)`,
		actorID, action, objectType, objectID, b, a,
	).Error
}

// Gin context keys populated by handlers on audited routes.
const (
	// CtxObjectID holds a uuid.UUID, *uuid.UUID, or string identifying the object.
	CtxObjectID = "audit.object_id"
	// CtxBefore / CtxAfter hold JSON-marshallable snapshots.
	CtxBefore = "audit.before"
	CtxAfter  = "audit.after"
)

// Middleware returns Gin middleware that appends an audit entry after a
// successful request (status < 400). The authenticated user (if any) is the
// actor; object_id/before/after are read from the gin context (set by the
// route handler). Audit failures are logged, never returned to the client.
func (s *Service) Middleware(action, objectType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Status() >= 400 {
			return
		}

		var actorID *uuid.UUID
		if u := auth.CurrentUser(c); u != nil {
			id := u.ID
			actorID = &id
		}

		var objectID *uuid.UUID
		if v, ok := c.Get(CtxObjectID); ok {
			switch id := v.(type) {
			case uuid.UUID:
				objectID = &id
			case *uuid.UUID:
				objectID = id
			case string:
				if parsed, err := uuid.Parse(id); err == nil {
					objectID = &parsed
				}
			}
		}

		var before, after any
		if v, ok := c.Get(CtxBefore); ok {
			before = v
		}
		if v, ok := c.Get(CtxAfter); ok {
			after = v
		}

		if err := s.Append(c.Request.Context(), actorID, action, objectType, objectID, before, after); err != nil {
			s.log.Error("audit append failed",
				"error", err,
				"action", action,
				"object_type", objectType,
				"object_id", objectID,
			)
		}
	}
}
