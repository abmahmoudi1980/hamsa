// Package notification provides the in-app notification channel (guaranteed)
// and the best-effort PushNotifier abstraction (research.md R10).
package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Notification mirrors the `notifications` table (migration 0001).
type Notification struct {
	ID        uuid.UUID  `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"column:user_id" json:"-"`
	Type      string     `gorm:"column:type" json:"type"`
	Title     string     `gorm:"column:title" json:"title"`
	Body      *string    `gorm:"column:body" json:"body,omitempty"`
	RefType   *string    `gorm:"column:ref_type" json:"ref_type,omitempty"`
	RefID     *uuid.UUID `gorm:"column:ref_id" json:"ref_id,omitempty"`
	IsRead    bool       `gorm:"column:is_read" json:"is_read"`
	ReadAt    *time.Time `gorm:"column:read_at" json:"read_at,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }

// PushNotifier is the best-effort push channel (research.md R10). In-app
// notifications remain the guaranteed channel when push is unavailable.
type PushNotifier interface {
	Push(ctx context.Context, userID uuid.UUID, title, body string) error
}

// NoopNotifier drops pushes.
type NoopNotifier struct{}

func (NoopNotifier) Push(context.Context, uuid.UUID, string, string) error { return nil }

// Service stores and queries in-app notifications.
type Service struct {
	db     *gorm.DB
	pusher PushNotifier
}

// NewService returns a notification service. A nil pusher defaults to the
// no-op notifier.
func NewService(db *gorm.DB, pusher PushNotifier) *Service {
	if pusher == nil {
		pusher = NoopNotifier{}
	}
	return &Service{db: db, pusher: pusher}
}

// Create stores an in-app notification and best-effort pushes it.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, typ, title, body string, refType *string, refID *uuid.UUID) (*Notification, error) {
	n := &Notification{
		ID:      uuid.New(),
		UserID:  userID,
		Type:    typ,
		Title:   title,
		Body:    &body,
		RefType: refType,
		RefID:   refID,
		IsRead:  false,
	}
	if err := s.db.WithContext(ctx).Create(n).Error; err != nil {
		return nil, err
	}
	_ = s.pusher.Push(ctx, userID, title, body) // best-effort
	return n, nil
}

// List returns a page of notifications for a user, newest first.
func (s *Service) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, page, pageSize int) ([]Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	q := s.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ?", userID)
	if unreadOnly {
		q = q.Where("is_read = ?", false)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []Notification
	if err := q.Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// MarkRead marks one notification read, scoped to the user (idempotent).
func (s *Service) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ? AND user_id = ? AND is_read = ?", id, userID, false).
		Updates(map[string]any{"is_read": true, "read_at": now}).Error
}

// MarkAllRead marks every notification read for a user.
func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]any{"is_read": true, "read_at": now}).Error
}
