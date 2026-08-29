package billing

// T046 — GORM repositories for billing periods, cost items, shares,
// invoices, items, and adjustments (data-model.md), plus the DB reads the
// calculation service needs (units, occupant counts as-of, balance snapshot
// delegation).

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"hamsa/internal/building"
)

// Repository provides billing persistence.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a billing repository backed by db.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// --- periods --------------------------------------------------------------------

// CreatePeriod inserts a billing period.
func (r *Repository) CreatePeriod(ctx context.Context, p *BillingPeriod) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// GetPeriod returns one period or gorm.ErrRecordNotFound.
func (r *Repository) GetPeriod(ctx context.Context, id uuid.UUID) (*BillingPeriod, error) {
	var p BillingPeriod
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPeriods returns a building's periods, newest first.
func (r *Repository) ListPeriods(ctx context.Context, buildingID uuid.UUID) ([]BillingPeriod, error) {
	var out []BillingPeriod
	err := r.db.WithContext(ctx).
		Where("building_id = ?", buildingID).
		Order("created_at DESC").
		Find(&out).Error
	return out, err
}

// SavePeriod persists period field changes (status machine timestamps).
func (r *Repository) SavePeriod(ctx context.Context, p *BillingPeriod) error {
	return r.db.WithContext(ctx).Save(p).Error
}

// --- cost items -----------------------------------------------------------------

// GetCostItem returns one cost item or gorm.ErrRecordNotFound.
func (r *Repository) GetCostItem(ctx context.Context, id uuid.UUID) (*CostItem, error) {
	var ci CostItem
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&ci).Error; err != nil {
		return nil, err
	}
	return &ci, nil
}

// ListCostItems returns a period's cost items in insertion order.
func (r *Repository) ListCostItems(ctx context.Context, periodID uuid.UUID) ([]CostItem, error) {
	var out []CostItem
	err := r.db.WithContext(ctx).Where("period_id = ?", periodID).Order("created_at, id").Find(&out).Error
	return out, err
}

// IsManagerOf reports whether the user holds a user_buildings grant for the
// building (FR-037 manager scope).
func (r *Repository) IsManagerOf(ctx context.Context, userID, buildingID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&building.UserBuilding{}).
		Where("user_id = ? AND building_id = ?", userID, buildingID).
		Count(&n).Error
	return n > 0, err
}

// BuildingByID returns a building by id (scope checks handled by callers).
func (r *Repository) BuildingByID(ctx context.Context, id uuid.UUID) (*building.Building, error) {
	var b building.Building
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// CreateCostItem inserts a cost item. Nil JSON-array fields are normalized
// to empty (the unit_ids column is NOT NULL).
func (r *Repository) CreateCostItem(ctx context.Context, ci *CostItem) error {
	if ci.UnitIDs == nil {
		ci.UnitIDs = []uuid.UUID{}
	}
	if ci.ComboWeights == nil {
		ci.ComboWeights = []ComboWeight{}
	}
	return r.db.WithContext(ctx).Create(ci).Error
}

// SaveCostItem persists cost-item edits (draft periods only — service rule).
func (r *Repository) SaveCostItem(ctx context.Context, ci *CostItem) error {
	if ci.UnitIDs == nil {
		ci.UnitIDs = []uuid.UUID{}
	}
	if ci.ComboWeights == nil {
		ci.ComboWeights = []ComboWeight{}
	}
	return r.db.WithContext(ctx).Save(ci).Error
}

// DeleteCostItem removes a draft cost item; its shares cascade.
func (r *Repository) DeleteCostItem(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&CostItem{}).Error
}

// --- calculation inputs -----------------------------------------------------------

// UnitsForBuilding returns the building's live units ordered by number.
func (r *Repository) UnitsForBuilding(ctx context.Context, buildingID uuid.UUID) ([]building.Unit, error) {
	var out []building.Unit
	err := r.db.WithContext(ctx).
		Where("building_id = ? AND deleted_at IS NULL", buildingID).
		Order("number").
		Find(&out).Error
	return out, err
}

// OccupantCountAsOf returns the occupant count effective at ref (max
// effective_from ≤ ref), or 0 when none recorded (data-model.md BR-04).
func (r *Repository) OccupantCountAsOf(ctx context.Context, unitID uuid.UUID, ref time.Time) (int, error) {
	var cnt int
	err := r.db.WithContext(ctx).Raw(
		`SELECT COALESCE((SELECT occupant_count FROM occupant_count_history
		   WHERE unit_id = ? AND effective_from <= ?::date
		   ORDER BY effective_from DESC LIMIT 1), 0)`,
		unitID, ref.Format("2006-01-02"),
	).Scan(&cnt).Error
	return cnt, err
}

// --- calculation output -----------------------------------------------------------

// InvoiceWithItems bundles a draft invoice with its append-only items for a
// single-transaction write.
type InvoiceWithItems struct {
	Invoice Invoice
	Items   []InvoiceItem
}

// ReplaceCalculation atomically stores a fresh calculation: deletes the
// period's previous shares and draft invoices (allowed only while the period
// is draft/calculated — the DB trigger blocks deletes after issuance), then
// inserts the new shares/invoices/items.
func (r *Repository) ReplaceCalculation(ctx context.Context, periodID uuid.UUID, shares []CostItemShare, invoices []InvoiceWithItems) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`DELETE FROM cost_item_shares WHERE cost_item_id IN
			   (SELECT id FROM cost_items WHERE period_id = ?)`, periodID,
		).Error; err != nil {
			return err
		}
		if err := tx.Where("period_id = ?", periodID).Delete(&Invoice{}).Error; err != nil {
			return err
		}
		for i := range shares {
			if err := tx.Create(&shares[i]).Error; err != nil {
				return err
			}
		}
		for i := range invoices {
			if err := tx.Create(&invoices[i].Invoice).Error; err != nil {
				return err
			}
			for j := range invoices[i].Items {
				invoices[i].Items[j].InvoiceID = invoices[i].Invoice.ID
				if err := tx.Create(&invoices[i].Items[j]).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// DiscardCalculation removes shares and draft invoices when a calculated
// period reopens (calculated → draft).
func (r *Repository) DiscardCalculation(ctx context.Context, periodID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`DELETE FROM cost_item_shares WHERE cost_item_id IN
			   (SELECT id FROM cost_items WHERE period_id = ?)`, periodID,
		).Error; err != nil {
			return err
		}
		return tx.Where("period_id = ?", periodID).Delete(&Invoice{}).Error
	})
}

// ListSharesForPeriod returns every calculated share of a period's items.
func (r *Repository) ListSharesForPeriod(ctx context.Context, periodID uuid.UUID) ([]CostItemShare, error) {
	var out []CostItemShare
	err := r.db.WithContext(ctx).Raw(
		`SELECT s.* FROM cost_item_shares s
		   JOIN cost_items c ON c.id = s.cost_item_id
		  WHERE c.period_id = ?`, periodID,
	).Scan(&out).Error
	return out, err
}

// --- invoices ---------------------------------------------------------------------

// InvoiceFilter is the manager invoice list filter (contracts/api.md).
type InvoiceFilter struct {
	BuildingID uuid.UUID
	PeriodID   *uuid.UUID
	UnitID     *uuid.UUID
	Status     string
	Page       int
	PageSize   int
}

// ListInvoices returns one filtered page of a building's invoices plus total.
func (r *Repository) ListInvoices(ctx context.Context, f InvoiceFilter) ([]Invoice, int64, error) {
	q := r.db.WithContext(ctx).Model(&Invoice{}).Where("building_id = ?", f.BuildingID)
	if f.PeriodID != nil {
		q = q.Where("period_id = ?", *f.PeriodID)
	}
	if f.UnitID != nil {
		q = q.Where("unit_id = ?", *f.UnitID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Invoice
	page, size := normalizePage(f.Page, f.PageSize)
	err := q.
		Select(`invoices.*, (SELECT number FROM units u WHERE u.id = invoices.unit_id) AS unit_number,
		        (SELECT title FROM billing_periods bp WHERE bp.id = invoices.period_id) AS period_title`).
		Order("invoices.created_at DESC, invoices.id").
		Limit(size).Offset((page - 1) * size).
		Scan(&out).Error
	return out, total, err
}

// GetInvoice returns one invoice with items, adjustments, and display fields.
func (r *Repository) GetInvoice(ctx context.Context, id uuid.UUID) (*Invoice, error) {
	var inv Invoice
	err := r.db.WithContext(ctx).Model(&Invoice{}).
		Select(`invoices.*, (SELECT number FROM units u WHERE u.id = invoices.unit_id) AS unit_number,
		        (SELECT title FROM billing_periods bp WHERE bp.id = invoices.period_id) AS period_title`).
		Where("invoices.id = ?", id).
		Scan(&inv).Error
	if err != nil {
		return nil, err
	}
	if inv.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	if err := r.db.WithContext(ctx).Where("invoice_id = ?", id).Order("created_at, id").Find(&inv.Items).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("invoice_id = ?", id).Order("created_at").Find(&inv.Adjustments).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

var clauseForUpdate = clause.Locking{Strength: "UPDATE"}

// ListInvoicesForUnits returns a page of invoices for the given unit ids
// (resident /me/invoices scope), newest first.
func (r *Repository) ListInvoicesForUnits(ctx context.Context, unitIDs []uuid.UUID, page, size int) ([]Invoice, int64, error) {
	if len(unitIDs) == 0 {
		return []Invoice{}, 0, nil
	}
	q := r.db.WithContext(ctx).Model(&Invoice{}).Where("unit_id IN ?", unitIDs)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []Invoice
	page, size = normalizePage(page, size)
	err := q.
		Select(`invoices.*, (SELECT number FROM units u WHERE u.id = invoices.unit_id) AS unit_number,
		        (SELECT title FROM billing_periods bp WHERE bp.id = invoices.period_id) AS period_title`).
		Order("invoices.created_at DESC, invoices.id").
		Limit(size).Offset((page - 1) * size).
		Scan(&out).Error
	return out, total, err
}

// IssueInvoices atomically numbers and freezes every draft invoice of the
// period: SELECT ... FOR UPDATE on the period row serializes issuance; the
// building's live invoice count seeds the sequential numbering.
func (r *Repository) IssueInvoices(ctx context.Context, periodID uuid.UUID, year int, now time.Time) ([]Invoice, error) {
	var issued []Invoice
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p BillingPeriod
		if err := tx.Clauses(clauseForUpdate).
			Where("id = ?", periodID).First(&p).Error; err != nil {
			return err
		}
		if p.Status != PeriodCalculated {
			return ErrNotCalculable
		}
		var count int64
		if err := tx.Model(&Invoice{}).
			Where("building_id = ? AND invoice_number NOT LIKE 'DRAFT-%'", p.BuildingID).
			Count(&count).Error; err != nil {
			return err
		}
		var drafts []Invoice
		if err := tx.Where("period_id = ?", periodID).Order("unit_id").Find(&drafts).Error; err != nil {
			return err
		}
		for i := range drafts {
			seq := count + int64(i) + 1
			if err := tx.Model(&Invoice{}).Where("id = ?", drafts[i].ID).Updates(map[string]any{
				"invoice_number":   fmt.Sprintf("BLD-%d-%04d", year, seq),
				"issue_date":       now.Format("2006-01-02"),
				"inputs_frozen_at": now,
				"updated_at":       now,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&BillingPeriod{}).Where("id = ?", periodID).Updates(map[string]any{
			"status":     PeriodIssued,
			"issued_at":  now,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		// Re-read numbered invoices for notifications/response.
		return tx.Where("period_id = ?", periodID).Find(&issued).Error
	})
	if err != nil {
		return nil, err
	}
	return issued, nil
}

// SetInvoiceStatus updates only the status column (payment/cancel machine);
// the DB trigger rejects any amount mutation.
func (r *Repository) SetInvoiceStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&Invoice{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now()}).Error
}

// CreateAdjustment appends an adjustment row and its invoice item atomically
// (FR-017): the original amounts are never touched.
func (r *Repository) CreateAdjustment(ctx context.Context, adj *InvoiceAdjustment, item *InvoiceItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(adj).Error; err != nil {
			return err
		}
		item.AdjustmentID = &adj.ID
		return tx.Create(item).Error
	})
}

// --- resident notification lookup --------------------------------------------------

// OccupantUserIDs returns the distinct user ids whose phone matches an active
// occupant (any relationship) of the given units — recipients of
// invoice_issued notifications.
func (r *Repository) OccupantUserIDs(ctx context.Context, unitIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(unitIDs) == 0 {
		return nil, nil
	}
	var out []uuid.UUID
	err := r.db.WithContext(ctx).Raw(
		`SELECT DISTINCT u.id FROM users u
		   JOIN persons p ON p.phone = u.phone AND p.deleted_at IS NULL
		   JOIN occupancies o ON o.person_id = p.id AND o.end_date IS NULL
		  WHERE o.unit_id IN ? AND u.deleted_at IS NULL`, unitIDs,
	).Scan(&out).Error
	return out, err
}

// --- helpers ------------------------------------------------------------------------

var ErrNotCalculable = errors.New("دوره در وضعیت محاسبه نیست")

func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}
