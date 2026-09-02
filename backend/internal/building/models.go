// Package building implements US2 (buildings & units registry) and hosts the
// US3 people/occupancy domain (added by later tasks in this package).
package building

import (
	"time"

	"github.com/google/uuid"
)

// Unit status values (migration 0002 unit_status enum).
const (
	UnitStatusActive   = "active"
	UnitStatusVacant   = "vacant"
	UnitStatusOccupied = "occupied"
	UnitStatusInactive = "inactive"
)

// Building is the manager-registered building (data-model.md "buildings").
type Building struct {
	ID             uuid.UUID  `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	Name           string     `gorm:"column:name" json:"name"`
	Address        *string    `gorm:"column:address" json:"address"`
	BlockCount     int        `gorm:"column:block_count" json:"block_count"`
	FloorCount     int        `gorm:"column:floor_count" json:"floor_count"`
	UnitCount      int        `gorm:"column:unit_count" json:"unit_count"`
	BuiltYear      *int       `gorm:"column:built_year" json:"built_year"`
	ManagerPhone   *string    `gorm:"column:manager_phone" json:"manager_phone"`
	EmergencyPhone *string    `gorm:"column:emergency_phone" json:"emergency_phone"`
	Notes          *string    `gorm:"column:notes" json:"notes"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt      *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (Building) TableName() string { return "buildings" }

// Unit is a building unit (data-model.md "units"). Soft delete preserves
// financial history.
type Unit struct {
	ID             uuid.UUID  `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	BuildingID     uuid.UUID  `gorm:"column:building_id;type:uuid" json:"building_id"`
	Number         string     `gorm:"column:number" json:"number"`
	Block          *string    `gorm:"column:block" json:"block"`
	Floor          int        `gorm:"column:floor" json:"floor"`
	AreaM2         int        `gorm:"column:area_m2" json:"area_m2"`
	ParkingCount   int        `gorm:"column:parking_count" json:"parking_count"`
	ParkingNumbers *string    `gorm:"column:parking_numbers" json:"parking_numbers"`
	StorageCount   int        `gorm:"column:storage_count" json:"storage_count"`
	StorageNumbers *string    `gorm:"column:storage_numbers" json:"storage_numbers"`
	Status         string     `gorm:"column:status" json:"status"`
	Notes          *string    `gorm:"column:notes" json:"notes"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt      *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (Unit) TableName() string { return "units" }

// UserBuilding is one row of the manager permission scope
// (data-model.md "user_buildings").
type UserBuilding struct {
	UserID     uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	BuildingID uuid.UUID `gorm:"column:building_id;type:uuid;primaryKey"`
	GrantedAt  time.Time `gorm:"column:granted_at"`
}

func (UserBuilding) TableName() string { return "user_buildings" }

// BuildingManager is one row of a building's manager list — the join of
// user_buildings with the manager's identity (002 data-model.md
// "BuildingManagerView"). Role is the account role (always manager for
// grantees; carried for client display and the grant response).
type BuildingManager struct {
	UserID    uuid.UUID `gorm:"column:user_id" json:"user_id"`
	Phone     string    `gorm:"column:phone" json:"phone"`
	Name      string    `gorm:"column:name" json:"name"`
	Role      string    `gorm:"column:role" json:"role"`
	GrantedAt time.Time `gorm:"column:granted_at" json:"granted_at"`
}
