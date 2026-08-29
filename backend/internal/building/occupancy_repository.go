package building

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// ErrOccupancyConflict is returned when a concurrent active-occupancy insert
// violates uq_occupancies_active_tenant — mapped to 409 by the service layer.
var ErrOccupancyConflict = errors.New("occupancy conflict")

// Repository adds US3 persistence (persons, occupancies, occupant counts)
// on top of the shared gorm handle. Occupancy rows are end-dated, never
// deleted (FR-007).
func (r *Repository) CreatePerson(ctx context.Context, p *Person) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// FindPerson returns a live person by id, or gorm.ErrRecordNotFound.
func (r *Repository) FindPerson(ctx context.Context, id uuid.UUID) (*Person, error) {
	var p Person
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPersons returns one page of a building's live persons (optional q on
// name/phone/national_id), newest first, plus the total count.
func (r *Repository) ListPersons(ctx context.Context, buildingID uuid.UUID, q string, page, size int) ([]Person, int64, error) {
	scope := r.db.WithContext(ctx).Model(&Person{}).
		Where("building_id = ? AND deleted_at IS NULL", buildingID)
	if q != "" {
		like := "%" + q + "%"
		scope = scope.Where("full_name ILIKE ? OR phone ILIKE ? OR national_id ILIKE ?", like, like, like)
	}
	var total int64
	if err := scope.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Person
	err := scope.Order("created_at DESC").Limit(size).Offset((page - 1) * size).Find(&items).Error
	return items, total, err
}

// UpdatePerson persists changed fields on a person.
func (r *Repository) UpdatePerson(ctx context.Context, p *Person) error {
	return r.db.WithContext(ctx).Save(p).Error
}

// DeletePerson soft-deletes (archives) a person. Occupancy history is kept.
func (r *Repository) DeletePerson(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Person{}, "id = ?", id).Error
}

// --- occupancies --------------------------------------------------------------

// CreateOccupancy inserts a new occupancy and, in the same transaction,
// end-dates the previous active occupancy with the SAME relationship on the
// unit (FR-007: "new active occupancy for a unit with the same relationship
// closes the previous one"). The closed row keeps start_date; its end_date
// is the day before the new start (never before its own start_date).
func (r *Repository) CreateOccupancy(ctx context.Context, o *Occupancy) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(
			`UPDATE occupancies
			    SET end_date = GREATEST(start_date, ?::date - 1), is_active = FALSE
			  WHERE unit_id = ? AND relationship = ? AND end_date IS NULL`,
			o.StartDate.Time, o.UnitID, o.Relationship)
		if res.Error != nil {
			return res.Error
		}
		if err := tx.Create(o).Error; err != nil {
			if isUniqueViolation(err) {
				return ErrOccupancyConflict
			}
			return err
		}
		return nil
	})
}

// isUniqueViolation reports a PostgreSQL 23505 unique-index violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

// FindOccupancy returns an occupancy by id, or gorm.ErrRecordNotFound.
func (r *Repository) FindOccupancy(ctx context.Context, id uuid.UUID) (*Occupancy, error) {
	var o Occupancy
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// ListOccupancies returns the full (historical) occupancy list of a unit,
// newest start first, with the person preloaded.
func (r *Repository) ListOccupancies(ctx context.Context, unitID uuid.UUID) ([]Occupancy, error) {
	var items []Occupancy
	err := r.db.WithContext(ctx).
		Preload("Person").
		Where("unit_id = ?", unitID).
		Order("start_date DESC, created_at DESC").
		Find(&items).Error
	return items, err
}

// EndOccupancy end-dates an occupancy row (preserving it, FR-007).
func (r *Repository) EndOccupancy(ctx context.Context, id uuid.UUID, end Date) error {
	return r.db.WithContext(ctx).Model(&Occupancy{}).
		Where("id = ?", id).
		Updates(map[string]any{"end_date": end, "is_active": false}).Error
}

// --- occupant counts ----------------------------------------------------------

// RecordOccupantCount appends one occupant-count data point (FR-008).
func (r *Repository) RecordOccupantCount(ctx context.Context, oc *OccupantCount) error {
	return r.db.WithContext(ctx).Create(oc).Error
}

// ListOccupantCounts returns the full occupant-count history of a unit,
// newest effective_from first.
func (r *Repository) ListOccupantCounts(ctx context.Context, unitID uuid.UUID) ([]OccupantCount, error) {
	var items []OccupantCount
	err := r.db.WithContext(ctx).
		Where("unit_id = ?", unitID).
		Order("effective_from DESC, created_at DESC").
		Find(&items).Error
	return items, err
}

// OccupantCountAsOf returns the count with max effective_from ≤ ref
// (ties broken by latest creation) — the value the charge engine snapshots
// (data-model.md). No data point yet ⇒ 0.
func (r *Repository) OccupantCountAsOf(ctx context.Context, unitID uuid.UUID, ref time.Time) (int, error) {
	var count int
	err := r.db.WithContext(ctx).Raw(
		`SELECT occupant_count
		   FROM occupant_count_history
		  WHERE unit_id = ? AND effective_from <= ?::date
		  ORDER BY effective_from DESC, created_at DESC
		  LIMIT 1`, unitID, ref.UTC()).Scan(&count).Error
	return count, err
}
