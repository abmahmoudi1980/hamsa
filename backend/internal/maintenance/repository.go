package maintenance

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/building"
)

// Repository provides maintenance persistence and scope helpers.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a maintenance repository backed by db.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// IsManagerOf reports whether the user holds a user_buildings grant for the building (FR-037).
func (r *Repository) IsManagerOf(ctx context.Context, userID, buildingID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&building.UserBuilding{}).
		Where("user_id = ? AND building_id = ?", userID, buildingID).
		Count(&n).Error
	return n > 0, err
}

// PhotoPath resolves a files-registry id (T010) to its storage path; gorm.ErrRecordNotFound when missing.
func (r *Repository) PhotoPath(ctx context.Context, fileID uuid.UUID) (string, error) {
	var f struct {
		Path string `gorm:"column:path"`
	}
	err := r.db.WithContext(ctx).Raw(`SELECT path FROM files WHERE id = ?`, fileID).Scan(&f).Error
	if err != nil {
		return "", err
	}
	if f.Path == "" {
		return "", gorm.ErrRecordNotFound
	}
	return f.Path, nil
}

// Create inserts a maintenance request.
func (r *Repository) Create(ctx context.Context, m *MaintenanceRequest) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// Get returns one request by id or gorm.ErrRecordNotFound.
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*MaintenanceRequest, error) {
	var m MaintenanceRequest
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// Save persists mutations (updated_at implicitly via GORM, closed_at managed by service).
func (r *Repository) Save(ctx context.Context, m *MaintenanceRequest) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// Filter narrows the manager's maintenance list.
type Filter struct {
	BuildingID uuid.UUID
	Status     string
	Priority   string
	Category   string
	Page       int
	Size       int
}

// ListForBuilding returns one page of a building's requests with total count (newest first).
func (r *Repository) ListForBuilding(ctx context.Context, f Filter) ([]MaintenanceRequest, int64, error) {
	q := r.db.WithContext(ctx).Model(&MaintenanceRequest{}).Where("building_id = ?", f.BuildingID)
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Priority != "" {
		q = q.Where("priority = ?", f.Priority)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []MaintenanceRequest
	err := q.Order("created_at DESC").Limit(f.Size).Offset((f.Page - 1) * f.Size).Find(&items).Error
	return items, total, err
}

// ListForUser returns one page of a user's own requests (resident view), newest first.
func (r *Repository) ListForUser(ctx context.Context, userID uuid.UUID, page, size int) ([]MaintenanceRequest, int64, error) {
	q := r.db.WithContext(ctx).Model(&MaintenanceRequest{}).Where("submitted_by = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []MaintenanceRequest
	err := q.Order("created_at DESC").Limit(size).Offset((page - 1) * size).Find(&items).Error
	return items, total, err
}

// ResolveResidentUnit returns the first active unit for the resident (via persons.phone → occupancies).
// Returns uuid.Nil + false when the resident has no active occupancy.
func (r *Repository) ResolveResidentUnit(ctx context.Context, phone string) (buildingID, unitID uuid.UUID, ok bool, err error) {
	type row struct {
		BuildingID uuid.UUID `gorm:"column:building_id"`
		UnitID     uuid.UUID `gorm:"column:unit_id"`
	}
	var res row
	// Join persons (phone) → occupancies (active) → units to obtain building + unit.
	err = r.db.WithContext(ctx).Raw(`
		SELECT u.building_id AS building_id, o.unit_id AS unit_id
		FROM persons p
		JOIN occupancies o ON o.person_id = p.id AND o.end_date IS NULL
		JOIN units u ON u.id = o.unit_id AND u.deleted_at IS NULL
		WHERE p.phone = ?
		ORDER BY o.created_at DESC
		LIMIT 1
	`, phone).Scan(&res).Error
	if err != nil {
		return uuid.Nil, uuid.Nil, false, err
	}
	if res.UnitID == uuid.Nil {
		return uuid.Nil, uuid.Nil, false, nil
	}
	return res.BuildingID, res.UnitID, true, nil
}

// UnitBuildingID returns the building id for a unit (or uuid.Nil when not found).
func (r *Repository) UnitBuildingID(ctx context.Context, unitID uuid.UUID) (uuid.UUID, error) {
	var bID uuid.UUID
	err := r.db.WithContext(ctx).Raw(`SELECT building_id FROM units WHERE id = ?`, unitID).Scan(&bID).Error
	return bID, err
}

// PersonExists reports whether a live person with id exists (for assignee validation).
func (r *Repository) PersonExists(ctx context.Context, personID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM persons WHERE id = ? AND deleted_at IS NULL`, personID).Scan(&n).Error
	return n > 0, err
}
