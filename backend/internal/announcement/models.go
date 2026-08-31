package announcement

import (
	"time"

	"github.com/google/uuid"
)

// Audience type constants (migration 0008 announcement_audience enum).
const (
	AudienceAll   = "all"
	AudienceBlock = "block"
	AudienceFloor = "floor"
	AudienceUnit  = "unit"
)

// Announcement is the building announcement row (data-model.md "announcements").
type Announcement struct {
	ID            uuid.UUID  `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	BuildingID    uuid.UUID  `gorm:"column:building_id;type:uuid" json:"building_id"`
	Title         string     `gorm:"column:title" json:"title"`
	Body          string     `gorm:"column:body" json:"body"`
	AudienceType  string     `gorm:"column:audience_type" json:"audience_type"`
	AudienceValue *string    `gorm:"column:audience_value" json:"audience_value"`
	PublishAt     *time.Time `gorm:"column:publish_at" json:"publish_at"`
	ExpireAt      *time.Time `gorm:"column:expire_at" json:"expire_at"`
	AttachmentFile *string   `gorm:"column:attachment_file" json:"attachment_file"`
	CreatedBy     *uuid.UUID `gorm:"column:created_by;type:uuid" json:"created_by"`
	CreatedAt     time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (Announcement) TableName() string { return "announcements" }

// AnnouncementRead tracks per-user read state (data-model.md "announcement_reads").
type AnnouncementRead struct {
	AnnouncementID uuid.UUID `gorm:"column:announcement_id;type:uuid;primaryKey"`
	UserID         uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	ReadAt         time.Time `gorm:"column:read_at"`
}

func (AnnouncementRead) TableName() string { return "announcement_reads" }

// AnnouncementWithRead enriches an announcement with the caller's read flag.
type AnnouncementWithRead struct {
	Announcement
	IsRead bool       `json:"is_read"`
	ReadAt *time.Time `json:"read_at"`
}
