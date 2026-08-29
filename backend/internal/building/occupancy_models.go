package building

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Occupancy relationship values (migration 0003 occupancy_relationship enum).
const (
	RelOwner            = "owner"
	RelTenant           = "tenant"
	RelNonResidentOwner = "non_resident_owner"
)

// Date is a PostgreSQL DATE column serialized as ISO-8601 "2006-01-02" on
// the wire (contracts/api.md: dates are ISO-8601; Jalali is presentation-only).
type Date struct{ time.Time }

// MarshalJSON renders the date as "YYYY-MM-DD".
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time.Format("2006-01-02"))
}

// UnmarshalJSON accepts "YYYY-MM-DD" only.
func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("date must be YYYY-MM-DD: %w", err)
	}
	d.Time = t.UTC()
	return nil
}

// Value binds the date as a time.Time so pgx casts it to DATE cleanly.
func (d Date) Value() (driver.Value, error) { return d.Time, nil }

// Scan reads a DATE column into the Date wrapper.
func (d *Date) Scan(v any) error {
	if v == nil {
		d.Time = time.Time{}
		return nil
	}
	switch t := v.(type) {
	case time.Time:
		d.Time = t
	case []byte:
		parsed, err := time.Parse("2006-01-02", string(t))
		if err != nil {
			return err
		}
		d.Time = parsed
	case string:
		parsed, err := time.Parse("2006-01-02", t)
		if err != nil {
			return err
		}
		d.Time = parsed
	default:
		return fmt.Errorf("unsupported date scan type %T", v)
	}
	return nil
}

// GormDataType tells GORM to use the DATE column type.
func (Date) GormDataType() string { return "date" }

// Person is a natural person linked to units (data-model.md "persons"),
// scoped per building for P0 simplicity.
type Person struct {
	ID         uuid.UUID  `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	BuildingID uuid.UUID  `gorm:"column:building_id;type:uuid" json:"building_id"`
	FullName   string     `gorm:"column:full_name" json:"full_name"`
	Phone      *string    `gorm:"column:phone" json:"phone"`
	NationalID *string    `gorm:"column:national_id" json:"national_id"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt  *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (Person) TableName() string { return "persons" }

// Occupancy is the dated person↔unit link (data-model.md "occupancies").
// Records are end-dated, never deleted, when a resident changes (FR-007).
type Occupancy struct {
	ID           uuid.UUID `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	UnitID       uuid.UUID `gorm:"column:unit_id;type:uuid" json:"unit_id"`
	PersonID     uuid.UUID `gorm:"column:person_id;type:uuid" json:"person_id"`
	Relationship string    `gorm:"column:relationship" json:"relationship"`
	StartDate    Date      `gorm:"column:start_date" json:"start_date"`
	EndDate      *Date     `gorm:"column:end_date" json:"end_date"`
	IsActive     bool      `gorm:"column:is_active" json:"is_active"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`

	// Person is preloaded on list/detail for the UI.
	Person *Person `gorm:"foreignKey:PersonID" json:"person,omitempty"`
}

func (Occupancy) TableName() string { return "occupancies" }

// OccupantCount is one independent occupant-count data point
// (data-model.md "occupant_count_history", FR-008/BR-04).
type OccupantCount struct {
	ID            uuid.UUID `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	UnitID        uuid.UUID `gorm:"column:unit_id;type:uuid" json:"unit_id"`
	OccupantCount int       `gorm:"column:occupant_count" json:"occupant_count"`
	EffectiveFrom Date      `gorm:"column:effective_from" json:"effective_from"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

func (OccupantCount) TableName() string { return "occupant_count_history" }
