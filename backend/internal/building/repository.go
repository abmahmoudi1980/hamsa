package building

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Repository provides persistence for buildings, units, and the manager
// scope (user_buildings). Soft-deleted rows are excluded via deleted_at.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a repository backed by db.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// DB exposes the underlying handle for services composing transactions.
func (r *Repository) DB() *gorm.DB { return r.db }

// --- buildings ---------------------------------------------------------------

// CreateBuilding inserts a new building row (ID may be pre-set).
func (r *Repository) CreateBuilding(ctx context.Context, b *Building) error {
	return r.db.WithContext(ctx).Create(b).Error
}

// FindBuilding returns a non-deleted building by id, or gorm.ErrRecordNotFound.
func (r *Repository) FindBuilding(ctx context.Context, id uuid.UUID) (*Building, error) {
	var b Building
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// ListBuildingsForManager returns the permitted, non-deleted buildings of a
// manager (FR-037), newest first.
func (r *Repository) ListBuildingsForManager(ctx context.Context, userID uuid.UUID) ([]Building, error) {
	var out []Building
	err := r.db.WithContext(ctx).
		Joins("JOIN user_buildings ub ON ub.building_id = buildings.id AND ub.user_id = ?", userID).
		Order("buildings.created_at DESC").
		Find(&out).Error
	return out, err
}

// UpdateBuilding persists changed fields on a building.
func (r *Repository) UpdateBuilding(ctx context.Context, b *Building) error {
	return r.db.WithContext(ctx).Save(b).Error
}

// DeleteBuilding soft-deletes a building.
func (r *Repository) DeleteBuilding(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Building{}, "id = ?", id).Error
}

// CountUnits returns the live (non-deleted) unit count of a building.
func (r *Repository) CountUnits(ctx context.Context, buildingID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Unit{}).
		Where("building_id = ? AND deleted_at IS NULL", buildingID).
		Count(&n).Error
	return n, err
}

// GrantManager grants a user access to a building (idempotent).
func (r *Repository) GrantManager(ctx context.Context, userID, buildingID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where(UserBuilding{UserID: userID, BuildingID: buildingID}).
		FirstOrCreate(&UserBuilding{}).Error
}

// CanManagerAccess reports whether userID is granted buildingID.
func (r *Repository) CanManagerAccess(ctx context.Context, userID, buildingID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&UserBuilding{}).
		Where("user_id = ? AND building_id = ?", userID, buildingID).
		Count(&n).Error
	return n > 0, err
}

// --- units --------------------------------------------------------------------

// ErrDuplicateUnitNumber is returned when (building_id, number) already
// exists among live units — mapped to 409 by the service layer.
var ErrDuplicateUnitNumber = errors.New("duplicate unit number")

// CreateUnit inserts a unit; duplicate live numbers surface as
// [ErrDuplicateUnitNumber].
func (r *Repository) CreateUnit(ctx context.Context, u *Unit) error {
	err := r.db.WithContext(ctx).Create(u).Error
	return mapDup(err)
}

// FindUnit returns a live unit by id, or gorm.ErrRecordNotFound.
func (r *Repository) FindUnit(ctx context.Context, id uuid.UUID) (*Unit, error) {
	var u Unit
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// UnitFilter narrows ListUnits (contracts: ?q=&block=&floor=&status=).
type UnitFilter struct {
	Q     string // substring match on number / block / notes
	Block string

	Floor  *int
	Status string
	Page   int
	Size   int
}

// ListUnits returns one filtered, paginated page of units plus the total
// count across all pages.
func (r *Repository) ListUnits(ctx context.Context, buildingID uuid.UUID, f UnitFilter) ([]Unit, int64, error) {
	q := r.db.WithContext(ctx).Model(&Unit{}).
		Where("building_id = ? AND deleted_at IS NULL", buildingID)
	if f.Q != "" {
		pat := "%" + f.Q + "%"
		q = q.Where("(number ILIKE ? OR COALESCE(block,'') ILIKE ? OR COALESCE(notes,'') ILIKE ?)", pat, pat, pat)
	}
	if f.Block != "" {
		q = q.Where("block = ?", f.Block)
	}
	if f.Floor != nil {
		q = q.Where("floor = ?", *f.Floor)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, size := f.Page, f.Size
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var out []Unit
	err := q.Order("number ASC").Limit(size).Offset((page - 1) * size).Find(&out).Error
	return out, total, err
}

// UpdateUnit persists changed fields on a unit; duplicate live numbers
// surface as [ErrDuplicateUnitNumber].
func (r *Repository) UpdateUnit(ctx context.Context, u *Unit) error {
	err := r.db.WithContext(ctx).Save(u).Error
	return mapDup(err)
}

// DeleteUnit soft-deletes a unit.
func (r *Repository) DeleteUnit(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Unit{}, "id = ?", id).Error
}

// pgUniqueViolation is the PostgreSQL error code for unique-index violations.
const pgUniqueViolation = "23505"

// mapDup translates the uq_units_building_number_active violation into
// [ErrDuplicateUnitNumber]; anything else passes through unchanged.
func mapDup(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return ErrDuplicateUnitNumber
	}
	return err
}
