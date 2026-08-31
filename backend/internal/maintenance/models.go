// Package maintenance implements US7: resident maintenance requests with
// the status workflow new → under_review → in_progress → done → closed
// (early close from under_review/in_progress/done), assignee/cost/notes
// updates, per-transition notifications (FR-033) and audit entries (FR-038).
package maintenance

import (
	"time"

	"github.com/google/uuid"
)

// Category values (migration 0007 maintenance_category, spec §14).
const (
	CatElevator   = "elevator"
	CatUtilities  = "utilities"
	CatElectrical = "electrical"
	CatWater      = "water"
	CatCleaning   = "cleaning"
	CatCommonArea = "common_area"
	CatParking    = "parking"
	CatOther      = "other"
)

var validCategories = map[string]bool{
	CatElevator: true, CatUtilities: true, CatElectrical: true,
	CatWater: true, CatCleaning: true, CatCommonArea: true,
	CatParking: true, CatOther: true,
}

// Priority values (migration 0007 maintenance_priority).
const (
	PriNormal    = "normal"
	PriImportant = "important"
	PriUrgent    = "urgent"
)

var validPriorities = map[string]bool{
	PriNormal: true, PriImportant: true, PriUrgent: true,
}

// Status values (migration 0007 maintenance_status, data-model.md).
const (
	StatusNew         = "new"
	StatusUnderReview = "under_review"
	StatusInProgress  = "in_progress"
	StatusDone        = "done"
	StatusClosed      = "closed"
)

var validStatuses = map[string]bool{
	StatusNew: true, StatusUnderReview: true, StatusInProgress: true,
	StatusDone: true, StatusClosed: true,
}

// allowedTransitions enumerates the legal forward edges (FR-029). Early-close
// is the set under_review/in_progress/done → closed.
var allowedTransitions = map[string]map[string]bool{
	StatusNew:         {StatusUnderReview: true},
	StatusUnderReview: {StatusInProgress: true, StatusClosed: true},
	StatusInProgress:  {StatusDone: true, StatusClosed: true},
	StatusDone:        {StatusClosed: true},
	// StatusClosed: terminal — no outgoing edges.
}

// MaintenanceRequest is one row of maintenance_requests (data-model.md).
type MaintenanceRequest struct {
	ID               uuid.UUID  `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	BuildingID       uuid.UUID  `gorm:"column:building_id;type:uuid" json:"building_id"`
	UnitID           *uuid.UUID `gorm:"column:unit_id;type:uuid" json:"unit_id,omitempty"`
	SubmittedBy      uuid.UUID  `gorm:"column:submitted_by;type:uuid" json:"submitted_by"`
	Title            string     `gorm:"column:title" json:"title"`
	Category         string     `gorm:"column:category" json:"category"`
	Description      *string    `gorm:"column:description" json:"description,omitempty"`
	Location         *string    `gorm:"column:location" json:"location,omitempty"`
	PhotoFile        *string    `gorm:"column:photo_file" json:"photo_file,omitempty"`
	Priority         string     `gorm:"column:priority" json:"priority"`
	Status           string     `gorm:"column:status" json:"status"`
	AssigneePersonID *uuid.UUID `gorm:"column:assignee_person_id;type:uuid" json:"assignee_person_id,omitempty"`
	RecordedCost     *int64     `gorm:"column:recorded_cost" json:"recorded_cost,omitempty"`
	Notes            *string    `gorm:"column:notes" json:"notes,omitempty"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at"`
	ClosedAt         *time.Time `gorm:"column:closed_at" json:"closed_at,omitempty"`
}

func (MaintenanceRequest) TableName() string { return "maintenance_requests" }

// canTransition reports whether status s may move to next.
func canTransition(s, next string) bool {
	if m, ok := allowedTransitions[s]; ok {
		return m[next]
	}
	return false
}
