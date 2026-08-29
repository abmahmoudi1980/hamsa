// Package expense implements US6: categorized building expenses with
// receipts and an approval workflow, plus the financial report (FR-027).
package expense

import (
	"time"

	"github.com/google/uuid"

	"hamsa/internal/building"
)

// Category values (migration 0006 expense_category, spec §13).
const (
	CatWater       = "water"
	CatElectricity = "electricity"
	CatGas         = "gas"
	CatElevator    = "elevator"
	CatCleaning    = "cleaning"
	CatSecurity    = "security"
	CatRepair      = "repair"
	CatInsurance   = "insurance"
	CatEquipment   = "equipment"
	CatOther       = "other"
)

// Categories is the canonical category list for validation and UI mapping.
var Categories = []string{
	CatWater, CatElectricity, CatGas, CatElevator, CatCleaning,
	CatSecurity, CatRepair, CatInsurance, CatEquipment, CatOther,
}

// Approval statuses (migration 0006 expense_approval).
const (
	ApprovalPending  = "pending"
	ApprovalApproved = "approved"
	ApprovalRejected = "rejected"
)

// Expense is one recorded building expense (data-model.md "expenses").
// Soft-deleted rows keep their ledger history but vanish from lists and
// reports (spec §21).
type Expense struct {
	ID             uuid.UUID     `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	BuildingID     uuid.UUID     `gorm:"column:building_id;type:uuid" json:"building_id"`
	Title          string        `gorm:"column:title" json:"title"`
	Category       string        `gorm:"column:category" json:"category"`
	Amount         int64         `gorm:"column:amount" json:"amount"`
	ExpenseDate    building.Date `gorm:"column:expense_date" json:"expense_date"`
	Description    *string       `gorm:"column:description" json:"description,omitempty"`
	PayerPersonID  *uuid.UUID    `gorm:"column:payer_person_id;type:uuid" json:"payer_person_id,omitempty"`
	ReceiptFile    *string       `gorm:"column:receipt_file" json:"receipt_file,omitempty"`
	ApprovalStatus string        `gorm:"column:approval_status" json:"approval_status"`
	CreatedBy      *uuid.UUID    `gorm:"column:created_by;type:uuid" json:"created_by,omitempty"`
	CreatedAt      time.Time     `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time     `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt      *time.Time    `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (Expense) TableName() string { return "expenses" }

// FinancialReport is the FR-027 aggregation for one building: month-scoped
// income/expense/net plus building-wide totals. Money is Toman.
type FinancialReport struct {
	Month          string `json:"month"` // YYYY-MM as requested
	MonthlyIncome  int64  `json:"monthly_income"`
	MonthlyExpense int64  `json:"monthly_expense"`
	Net            int64  `json:"net"` // monthly_income − monthly_expense
	TotalDebt      int64  `json:"total_debt"`
	TotalPayments  int64  `json:"total_payments"`
	TotalExpenses  int64  `json:"total_expenses"`
}
