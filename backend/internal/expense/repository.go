package expense

// T064 — GORM repository for expenses and the FR-027 report aggregations.
// Soft-deleted expenses are excluded everywhere via deleted_at IS NULL.

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/building"
)

// Repository provides expense persistence + report aggregation.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns an expense repository backed by db.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// IsManagerOf reports whether the user holds a user_buildings grant for the
// building (FR-037 manager scope).
func (r *Repository) IsManagerOf(ctx context.Context, userID, buildingID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&building.UserBuilding{}).
		Where("user_id = ? AND building_id = ?", userID, buildingID).
		Count(&n).Error
	return n > 0, err
}

// ReceiptPath resolves a files-registry id (T010) to its storage path;
// gorm.ErrRecordNotFound when the file does not exist.
func (r *Repository) ReceiptPath(ctx context.Context, fileID uuid.UUID) (string, error) {
	var f struct {
		Path string `gorm:"column:path"`
	}
	err := r.db.WithContext(ctx).Raw(
		`SELECT path FROM files WHERE id = ?`, fileID,
	).Scan(&f).Error
	if err != nil {
		return "", err
	}
	if f.Path == "" {
		return "", gorm.ErrRecordNotFound
	}
	return f.Path, nil
}

// CreateExpense inserts an expense row.
func (r *Repository) CreateExpense(ctx context.Context, e *Expense) error {
	return r.db.WithContext(ctx).Create(e).Error
}

// GetExpense returns one non-deleted expense or gorm.ErrRecordNotFound.
func (r *Repository) GetExpense(ctx context.Context, id uuid.UUID) (*Expense, error) {
	var e Expense
	if err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// SaveExpense persists mutations (updates updated_at implicitly via GORM).
func (r *Repository) SaveExpense(ctx context.Context, e *Expense) error {
	return r.db.WithContext(ctx).Save(e).Error
}

// SoftDeleteExpense stamps deleted_at; the row is never removed (spec §21).
func (r *Repository) SoftDeleteExpense(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`UPDATE expenses SET deleted_at = now(), updated_at = now() WHERE id = ? AND deleted_at IS NULL`,
		id,
	).Error
}

// ExpenseFilter narrows the manager's expense list.
type ExpenseFilter struct {
	BuildingID uuid.UUID
	Category   string
	Approval   string
	From       *time.Time // expense_date >= From
	To         *time.Time // expense_date <= To
	Page       int
	Size       int
}

// ListExpenses returns one page of non-deleted expenses with the total count.
func (r *Repository) ListExpenses(ctx context.Context, f ExpenseFilter) ([]Expense, int64, error) {
	q := r.db.WithContext(ctx).Model(&Expense{}).
		Where("building_id = ? AND deleted_at IS NULL", f.BuildingID)
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.Approval != "" {
		q = q.Where("approval_status = ?", f.Approval)
	}
	if f.From != nil {
		q = q.Where("expense_date >= ?", f.From)
	}
	if f.To != nil {
		q = q.Where("expense_date <= ?", f.To)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Expense
	err := q.Order("expense_date DESC, created_at DESC").
		Limit(f.Size).Offset((f.Page - 1) * f.Size).
		Find(&items).Error
	return items, total, err
}

// --- financial report (FR-027) --------------------------------------------------

// monthWindow converts a YYYY-MM string into [start, end) date bounds.
func monthWindow(year, month int) (time.Time, time.Time) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return start, end
}

// Report aggregates the financial report for one building. Semantics:
//
//   - monthly_income: verified payments with paid_at inside the month
//     (reversed/failed rows never count).
//   - monthly_expense: non-deleted expenses with expense_date inside the
//     month (all approval statuses — recording is the financial fact).
//   - total_debt: Σ positive unit_balances.balance over the building's
//     non-deleted units (BR-01 rows are the single source of truth; credit
//     positions do not offset other units' debt).
//   - total_payments / total_expenses: building-wide verified payment and
//     non-deleted expense sums (all time).
func (r *Repository) Report(ctx context.Context, buildingID uuid.UUID, year, month int) (*FinancialReport, error) {
	start, end := monthWindow(year, month)
	rep := &FinancialReport{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(
			`SELECT COALESCE(SUM(amount), 0) FROM payments
			 WHERE building_id = ? AND status = 'verified' AND paid_at >= ? AND paid_at < ?`,
			buildingID, start, end,
		).Scan(&rep.MonthlyIncome).Error; err != nil {
			return err
		}
		if err := tx.Raw(
			`SELECT COALESCE(SUM(amount), 0) FROM expenses
			 WHERE building_id = ? AND deleted_at IS NULL AND expense_date >= ? AND expense_date < ?`,
			buildingID, start, end,
		).Scan(&rep.MonthlyExpense).Error; err != nil {
			return err
		}
		if err := tx.Raw(
			`SELECT COALESCE(SUM(b.balance), 0) FROM unit_balances b
			 JOIN units u ON u.id = b.unit_id
			 WHERE u.building_id = ? AND u.deleted_at IS NULL AND b.balance > 0`,
			buildingID,
		).Scan(&rep.TotalDebt).Error; err != nil {
			return err
		}
		if err := tx.Raw(
			`SELECT COALESCE(SUM(amount), 0) FROM payments
			 WHERE building_id = ? AND status = 'verified'`,
			buildingID,
		).Scan(&rep.TotalPayments).Error; err != nil {
			return err
		}
		return tx.Raw(
			`SELECT COALESCE(SUM(amount), 0) FROM expenses
			 WHERE building_id = ? AND deleted_at IS NULL`,
			buildingID,
		).Scan(&rep.TotalExpenses).Error
	})
	if err != nil {
		return nil, err
	}
	rep.Net = rep.MonthlyIncome - rep.MonthlyExpense
	return rep, nil
}
