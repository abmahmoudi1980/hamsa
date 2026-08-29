// Package billing implements US4: billing periods, cost items, the pure
// charge engine wiring (calculation), reviewable previews, and immutable
// invoices with explicit corrections. The engine itself lives in engine/
// with zero infrastructure imports.
package billing

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"hamsa/internal/billing/engine"
	"hamsa/internal/building"
)

// Money is an integer Toman amount serialized as a JSON string on the wire
// (contracts/api.md: amounts as strings to avoid client precision loss).
type Money int64

// MarshalJSON renders the amount as a quoted integer string.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(strconv.FormatInt(int64(m), 10))), nil
}

// UnmarshalJSON accepts a string or number.
func (m *Money) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("مبلغ نامعتبر است: %s", s)
		}
		*m = Money(v)
		return nil
	}
	var n int64
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("مبلغ نامعتبر است")
	}
	*m = Money(n)
	return nil
}

// Period status values (migration 0004 billing_period_status enum).
const (
	PeriodDraft      = "draft"
	PeriodCalculated = "calculated"
	PeriodIssued     = "issued"
	PeriodClosed     = "closed"
)

// Invoice status values (migration 0004 invoice_status enum).
const (
	InvoiceUnpaid   = "unpaid"
	InvoicePartial  = "partial"
	InvoicePaid     = "paid"
	InvoiceExpired  = "expired"
	InvoiceCanceled = "cancelled"
)

// BillingPeriod is a charge cycle (data-model.md "billing_periods").
type BillingPeriod struct {
	ID           uuid.UUID     `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	BuildingID   uuid.UUID     `gorm:"column:building_id;type:uuid" json:"building_id"`
	Title        string        `gorm:"column:title" json:"title"`
	StartDate    building.Date `gorm:"column:start_date" json:"start_date"`
	EndDate      building.Date `gorm:"column:end_date" json:"end_date"`
	DueDate      building.Date `gorm:"column:due_date" json:"due_date"`
	LateFeeType  string        `gorm:"column:late_fee_type" json:"late_fee_type"`
	LateFeeValue float64       `gorm:"column:late_fee_value" json:"late_fee_value"`
	Status       string        `gorm:"column:status" json:"status"`
	CalculatedAt *time.Time    `gorm:"column:calculated_at" json:"calculated_at,omitempty"`
	IssuedAt     *time.Time    `gorm:"column:issued_at" json:"issued_at,omitempty"`
	ClosedAt     *time.Time    `gorm:"column:closed_at" json:"closed_at,omitempty"`
	CreatedAt    time.Time     `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time     `gorm:"column:updated_at" json:"updated_at"`
}

func (BillingPeriod) TableName() string { return "billing_periods" }

// ComboWeight is one component of a combined cost method.
type ComboWeight struct {
	Method             string `json:"method"`
	Weight             int    `json:"weight"`
	FixedAmountPerUnit Money  `json:"fixed_amount_per_unit,omitempty"`
}

// CostItem is one cost line of a period (data-model.md "cost_items").
type CostItem struct {
	ID                 uuid.UUID     `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	PeriodID           uuid.UUID     `gorm:"column:period_id;type:uuid" json:"period_id"`
	Title              string        `gorm:"column:title" json:"title"`
	TotalAmount        Money         `gorm:"column:total_amount" json:"total_amount"`
	Method             string        `gorm:"column:method" json:"method"`
	FixedAmountPerUnit *Money        `gorm:"column:fixed_amount_per_unit" json:"fixed_amount_per_unit,omitempty"`
	ComboWeights       []ComboWeight `gorm:"column:combo_weights;serializer:json" json:"combo_weights,omitempty"`
	IncludeVacant      bool          `gorm:"column:include_vacant" json:"include_vacant"`
	UnitIDs            UnitIDList    `gorm:"column:unit_ids" json:"unit_ids,omitempty"`
	CreatedAt          time.Time     `gorm:"column:created_at" json:"created_at"`
	UpdatedAt          time.Time     `gorm:"column:updated_at" json:"updated_at"`
}

// UnitIDList maps a PostgreSQL UUID[] column (cost_items.unit_ids).
type UnitIDList []uuid.UUID

// Value renders the list as a Postgres array literal.
func (u UnitIDList) Value() (driver.Value, error) {
	parts := make([]string, len(u))
	for i, id := range u {
		parts[i] = id.String()
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

// Scan parses a Postgres UUID[] value.
func (u *UnitIDList) Scan(v any) error {
	if v == nil {
		*u = nil
		return nil
	}
	s, ok := v.(string)
	if !ok {
		if b, isB := v.([]byte); isB {
			s = string(b)
		} else {
			return fmt.Errorf("unit_ids: unsupported type %T", v)
		}
	}
	s = strings.Trim(s, "{}")
	if s == "" {
		*u = UnitIDList{}
		return nil
	}
	parts := strings.Split(s, ",")
	out := make(UnitIDList, 0, len(parts))
	for _, p := range parts {
		id, err := uuid.Parse(strings.Trim(p, `"`))
		if err != nil {
			return err
		}
		out = append(out, id)
	}
	*u = out
	return nil
}
func (CostItem) TableName() string { return "cost_items" }

// CostItemShare is the calculated share of one unit for one cost item
// (data-model.md "cost_item_shares", BR-08).
type CostItemShare struct {
	ID             uuid.UUID      `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	CostItemID     uuid.UUID      `gorm:"column:cost_item_id;type:uuid" json:"cost_item_id"`
	UnitID         uuid.UUID      `gorm:"column:unit_id;type:uuid" json:"unit_id"`
	ExactShare     string         `gorm:"column:exact_share" json:"exact_share"`
	RoundedShare   Money          `gorm:"column:rounded_share" json:"rounded_share"`
	InputsSnapshot map[string]any `gorm:"column:inputs_snapshot;serializer:json" json:"inputs_snapshot"`
}

func (CostItemShare) TableName() string { return "cost_item_shares" }

// Invoice is the immutable per-unit bill (data-model.md "invoices").
type Invoice struct {
	ID             uuid.UUID      `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	InvoiceNumber  string         `gorm:"column:invoice_number" json:"invoice_number"`
	BuildingID     uuid.UUID      `gorm:"column:building_id;type:uuid" json:"building_id"`
	PeriodID       uuid.UUID      `gorm:"column:period_id;type:uuid" json:"period_id"`
	UnitID         uuid.UUID      `gorm:"column:unit_id;type:uuid" json:"unit_id"`
	BaseAmount     Money          `gorm:"column:base_amount" json:"base_amount"`
	PriorDebt      Money          `gorm:"column:prior_debt" json:"prior_debt"`
	LateFeeAmount  Money          `gorm:"column:late_fee_amount" json:"late_fee_amount"`
	CreditAmount   Money          `gorm:"column:credit_amount" json:"credit_amount"`
	FinalAmount    Money          `gorm:"column:final_amount" json:"final_amount"`
	IssueDate      *building.Date `gorm:"column:issue_date" json:"issue_date,omitempty"`
	DueDate        *building.Date `gorm:"column:due_date" json:"due_date,omitempty"`
	Status         string         `gorm:"column:status" json:"status"`
	PaidAmount     Money          `gorm:"column:paid_amount" json:"paid_amount"`
	InputsFrozenAt *time.Time     `gorm:"column:inputs_frozen_at" json:"inputs_frozen_at,omitempty"`
	CreatedAt      time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at" json:"updated_at"`

	// Items/adjustments populated by repository detail reads (not a column).
	Items       []InvoiceItem       `gorm:"-" json:"items,omitempty"`
	Adjustments []InvoiceAdjustment `gorm:"-" json:"adjustments,omitempty"`
	// UnitNumber/PeriodTitle denormalized for list/detail display (not columns).
	UnitNumber  string `gorm:"-" json:"unit_number,omitempty"`
	PeriodTitle string `gorm:"-" json:"period_title,omitempty"`
}

func (Invoice) TableName() string { return "invoices" }

// InvoiceItemKind values (migration 0004 invoice_item_kind enum).
const (
	ItemCharge     = "charge"
	ItemLateFee    = "late_fee"
	ItemAdjustment = "adjustment"
)

// InvoiceItem is one append-only displayable row of an invoice.
type InvoiceItem struct {
	ID           uuid.UUID  `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	InvoiceID    uuid.UUID  `gorm:"column:invoice_id;type:uuid" json:"invoice_id"`
	Kind         string     `gorm:"column:kind" json:"kind"`
	Title        string     `gorm:"column:title" json:"title"`
	CostItemID   *uuid.UUID `gorm:"column:cost_item_id;type:uuid" json:"cost_item_id,omitempty"`
	Amount       Money      `gorm:"column:amount" json:"amount"`
	Method       *string    `gorm:"column:method" json:"method,omitempty"`
	AdjustmentID *uuid.UUID `gorm:"column:adjustment_id;type:uuid" json:"adjustment_id,omitempty"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (InvoiceItem) TableName() string { return "invoice_items" }

// Adjustment kind values (migration 0004 adjustment_kind enum).
const (
	AdjustmentDebit  = "debit"
	AdjustmentCredit = "credit"
)

// InvoiceAdjustment is an explicit correction note (FR-017/BR-03).
type InvoiceAdjustment struct {
	ID        uuid.UUID  `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	InvoiceID uuid.UUID  `gorm:"column:invoice_id;type:uuid" json:"invoice_id"`
	Kind      string     `gorm:"column:kind" json:"kind"`
	Amount    Money      `gorm:"column:amount" json:"amount"`
	Reason    string     `gorm:"column:reason" json:"reason"`
	CreatedBy *uuid.UUID `gorm:"column:created_by;type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (InvoiceAdjustment) TableName() string { return "invoice_adjustments" }

// --- engine conversion helpers -------------------------------------------------

// engineLateFeeType maps the stored late_fee_type to the engine enum.
func engineLateFeeType(s string) engine.LateFeeType {
	switch s {
	case "fixed":
		return engine.LateFeeFixed
	case "percent":
		return engine.LateFeePercent
	case "per_day":
		return engine.LateFeePerDay
	default:
		return engine.LateFeeNone
	}
}

// exactShareString renders a rational as a 4-decimal fixed-point string for
// the NUMERIC(20,4) column.
func exactShareString(r *big.Rat) string {
	return r.FloatString(4)
}
