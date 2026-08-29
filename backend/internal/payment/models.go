// Package payment implements US5: manual + gateway payments, partial
// payments with surplus credit, the transactionally recomputed unit balance
// (spec §9), and the provider-agnostic gateway wiring (research.md R5).
package payment

import (
	"time"

	"github.com/google/uuid"

	"hamsa/internal/building"
)

// Money is an integer Toman amount (same representation as billing.Money;
// kept as a local alias to avoid a cross-domain type dependency).
type Money = int64

// Payment method values (migration 0005 payment_method).
const (
	MethodManual  = "manual"
	MethodGateway = "gateway"
)

// Payment status values (migration 0005 payment_status). Manual payments are
// manager-confirmed and land directly in `verified`; gateway payments start
// `recorded` (pending verify) and are applied only after callback
// verification. Reversal is status-only — rows are never deleted (BR-02).
const (
	StatusRecorded = "recorded"
	StatusVerified = "verified"
	StatusFailed   = "failed"
	StatusReversed = "reversed"
)

// Payment is one manual or gateway transaction (data-model.md "payments").
type Payment struct {
	ID             uuid.UUID     `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	BuildingID     uuid.UUID     `gorm:"column:building_id;type:uuid" json:"building_id"`
	UnitID         uuid.UUID     `gorm:"column:unit_id;type:uuid" json:"unit_id"`
	InvoiceID      *uuid.UUID    `gorm:"column:invoice_id;type:uuid" json:"invoice_id,omitempty"`
	Method         string        `gorm:"column:method" json:"method"`
	Amount         Money         `gorm:"column:amount" json:"amount"`
	PaidAt         building.Date `gorm:"column:paid_at" json:"paid_at"`
	TrackingNumber *string       `gorm:"column:tracking_number" json:"tracking_number,omitempty"`
	Gateway        *string       `gorm:"column:gateway" json:"gateway,omitempty"`
	Authority      *string       `gorm:"column:authority" json:"authority,omitempty"`
	RecordedBy     *uuid.UUID    `gorm:"column:recorded_by;type:uuid" json:"recorded_by,omitempty"`
	Status         string        `gorm:"column:status" json:"status"`
	CreatedAt      time.Time     `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time     `gorm:"column:updated_at" json:"updated_at"`

	// UnitNumber denormalized for ledger display (not a column).
	UnitNumber string `gorm:"-" json:"unit_number,omitempty"`
}

func (Payment) TableName() string { return "payments" }

// UnitBalance is the unit's current financial position (data-model.md
// "unit_balances", spec §9). Recomputed transactionally on every financial
// event; credit_asset is the unit's surplus money (overpayment remainder)
// not yet consumed by an invoice — an asset outside the debt formula.
type UnitBalance struct {
	UnitID               uuid.UUID `gorm:"column:unit_id;type:uuid;primaryKey" json:"unit_id"`
	PriorDebt            int64     `gorm:"column:prior_debt" json:"prior_debt"`
	CurrentInvoiceAmount int64     `gorm:"column:current_invoice_amount" json:"current_invoice_amount"`
	LateFeeTotal         int64     `gorm:"column:late_fee_total" json:"late_fee_total"`
	Credit               int64     `gorm:"column:credit" json:"credit"`
	PaidTotal            int64     `gorm:"column:paid_total" json:"paid_total"`
	CreditAsset          int64     `gorm:"column:credit_asset" json:"credit_asset"`
	Balance              int64     `gorm:"column:balance" json:"balance"`
	UpdatedAt            time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (UnitBalance) TableName() string { return "unit_balances" }
