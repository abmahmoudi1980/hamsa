package billing

// T048 — period lifecycle state machine and corrections:
//
//	draft → calculated (engine run, T047) → issued (freeze) → closed
//	calculated → draft (reopen before issue; 409 after)
//	issued → draft forbidden
//	invoice cancel: row preserved (BR-10), audited
//	adjustments: debit/credit notes, originals untouched (FR-017/BR-03)

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/audit"
	"hamsa/internal/auth"
	"hamsa/internal/billing/engine"
	"hamsa/internal/building"
	"hamsa/internal/notification"
	"hamsa/internal/platform/httpx"
)

// PeriodService owns the period state machine, issuance, cancellation, and
// adjustments.
type PeriodService struct {
	repo  *Repository
	notif *notification.Service
	audit *audit.Service
}

// NewPeriodService returns the period lifecycle service.
func NewPeriodService(repo *Repository, notif *notification.Service, aud *audit.Service) *PeriodService {
	return &PeriodService{repo: repo, notif: notif, audit: aud}
}

// Persian validation messages (contracts/api.md — all user-facing text is
// Persian).
const (
	msgPeriodTitleRequired   = "عنوان دوره الزامی است"
	msgPeriodDatesInvalid    = "تاریخ‌های دوره نامعتبر است (پایان ≥ شروع، سررسید ≥ پایان)"
	msgPeriodNotFound        = "دوره صدور صورتحساب یافت نشد"
	msgPeriodNotDraft        = "ویرایش فقط در وضعیت پیش‌نویس مجاز است"
	msgItemTitleRequired     = "عنوان قلم هزینه الزامی است"
	msgItemAmountInvalid     = "مبلغ قلم هزینه باید بزرگ‌تر از صفر باشد"
	msgItemMethodInvalid     = "روش محاسبه نامعتبر است"
	msgInvoiceNotFound       = "صورتحساب یافت نشد"
	msgInvoiceNotCancellable = "این صورتحساب قابل ابطال نیست"
	msgAdjustReasonRequired  = "دلیل اصلاحیه الزامی است"
	msgAdjustAmountInvalid   = "مبلغ اصلاحیه باید بزرگ‌تر از صفر باشد"
	msgNoBuildingAccess      = "شما به این ساختمان دسترسی ندارید"
)

var validLateFeeTypes = map[string]bool{"none": true, "fixed": true, "percent": true, "per_day": true}

// PeriodInput is the create/update payload for a billing period.
type PeriodInput struct {
	Title        string        `json:"title" binding:"required"`
	StartDate    building.Date `json:"start_date" binding:"required"`
	EndDate      building.Date `json:"end_date" binding:"required"`
	DueDate      building.Date `json:"due_date" binding:"required"`
	LateFeeType  string        `json:"late_fee_type"`
	LateFeeValue float64       `json:"late_fee_value"`
}

// authorizeBuilding verifies the manager scope for a building (403 regardless
// of existence when not permitted).
func (s *PeriodService) authorizeBuilding(ctx context.Context, manager *auth.User, buildingID uuid.UUID) error {
	ok, err := s.repo.IsManagerOf(ctx, manager.ID, buildingID)
	if err != nil {
		return err
	}
	if !ok {
		return httpx.Forbidden(msgNoBuildingAccess)
	}
	return nil
}

// applyPeriodInput validates and copies period fields.
func applyPeriodInput(p *BillingPeriod, in *PeriodInput) error {
	if in.EndDate.Time.Before(in.StartDate.Time) || in.DueDate.Time.Before(in.EndDate.Time) {
		return httpx.BadRequest(msgPeriodDatesInvalid)
	}
	lf := in.LateFeeType
	if lf == "" {
		lf = "none"
	}
	if !validLateFeeTypes[lf] {
		return httpx.BadRequest(msgItemMethodInvalid)
	}
	p.Title = in.Title
	p.StartDate = in.StartDate
	p.EndDate = in.EndDate
	p.DueDate = in.DueDate
	p.LateFeeType = lf
	p.LateFeeValue = in.LateFeeValue
	return nil
}

// CreatePeriod validates and creates a billing period (draft) for a permitted
// building.
func (s *PeriodService) CreatePeriod(ctx context.Context, manager *auth.User, buildingID uuid.UUID, in PeriodInput) (*BillingPeriod, error) {
	if err := s.authorizeBuilding(ctx, manager, buildingID); err != nil {
		return nil, err
	}
	p := &BillingPeriod{
		ID:         uuid.New(),
		BuildingID: buildingID,
		Status:     PeriodDraft,
	}
	if err := applyPeriodInput(p, &in); err != nil {
		return nil, err
	}
	if err := s.repo.CreatePeriod(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ListPeriods returns a permitted building's periods.
func (s *PeriodService) ListPeriods(ctx context.Context, manager *auth.User, buildingID uuid.UUID) ([]BillingPeriod, error) {
	if err := s.authorizeBuilding(ctx, manager, buildingID); err != nil {
		return nil, err
	}
	return s.repo.ListPeriods(ctx, buildingID)
}

// GetPeriod returns a permitted period detail.
func (s *PeriodService) GetPeriod(ctx context.Context, manager *auth.User, id uuid.UUID) (*BillingPeriod, error) {
	p, err := s.getAuthorizedPeriod(ctx, manager, id)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// UpdatePeriod edits a period (only in draft — 409 otherwise).
func (s *PeriodService) UpdatePeriod(ctx context.Context, manager *auth.User, id uuid.UUID, in PeriodInput) (*BillingPeriod, *BillingPeriod, error) {
	p, err := s.getAuthorizedPeriod(ctx, manager, id)
	if err != nil {
		return nil, nil, err
	}
	if p.Status != PeriodDraft {
		return nil, nil, httpx.Conflict(msgPeriodNotDraft)
	}
	before := *p
	if err := applyPeriodInput(p, &in); err != nil {
		return nil, nil, err
	}
	if err := s.repo.SavePeriod(ctx, p); err != nil {
		return nil, nil, err
	}
	return p, &before, nil
}

// getAuthorizedPeriod loads a period and checks the manager scope.
func (s *PeriodService) getAuthorizedPeriod(ctx context.Context, manager *auth.User, id uuid.UUID) (*BillingPeriod, error) {
	p, err := s.repo.GetPeriod(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound(msgPeriodNotFound)
		}
		return nil, err
	}
	if err := s.authorizeBuilding(ctx, manager, p.BuildingID); err != nil {
		return nil, err
	}
	return p, nil
}

// --- cost items -----------------------------------------------------------------

// CostItemInput is the add/edit payload for a cost item (contracts/api.md).
type CostItemInput struct {
	Title              string        `json:"title" binding:"required"`
	TotalAmount        Money         `json:"total_amount"`
	Method             string        `json:"method" binding:"required"`
	FixedAmountPerUnit Money         `json:"fixed_amount_per_unit"`
	ComboWeights       []ComboWeight `json:"combo_weights"`
	IncludeVacant      bool          `json:"include_vacant"`
	UnitIDs            []uuid.UUID   `json:"unit_ids"`
}

// validateCostItem checks method-specific fields and derives total_amount for
// fixed items (fixed_amount_per_unit × eligible participants).
func (s *PeriodService) validateCostItem(ctx context.Context, ci *CostItem, in *CostItemInput, buildingID uuid.UUID) error {
	ci.Title = in.Title
	ci.Method = in.Method
	ci.IncludeVacant = in.IncludeVacant
	ci.ComboWeights = in.ComboWeights
	ci.UnitIDs = in.UnitIDs

	switch engine.Method(in.Method) {
	case engine.MethodEqual, engine.MethodPerOccupant, engine.MethodPerArea:
		if in.TotalAmount <= 0 {
			return httpx.BadRequest(msgItemAmountInvalid)
		}
		ci.TotalAmount = in.TotalAmount
		ci.FixedAmountPerUnit = nil
	case engine.MethodFixed:
		if in.FixedAmountPerUnit <= 0 {
			return httpx.BadRequest(msgItemAmountInvalid)
		}
		// Derive the nominal total from the per-unit amount and the current
		// participant count (informational; the engine distributes per-unit).
		units, err := s.repo.UnitsForBuilding(ctx, buildingID)
		if err != nil {
			return err
		}
		n := 0
		for _, u := range units {
			if u.Status == building.UnitStatusVacant && !in.IncludeVacant {
				continue
			}
			n++
		}
		ci.FixedAmountPerUnit = &in.FixedAmountPerUnit
		ci.TotalAmount = Money(int64(in.FixedAmountPerUnit) * int64(n))
	case engine.MethodSpecificUnits:
		if in.TotalAmount <= 0 {
			return httpx.BadRequest(msgItemAmountInvalid)
		}
		if len(in.UnitIDs) == 0 {
			return httpx.BadRequest(engine.ErrSpecificUnitsEmpty.Error())
		}
		units, err := s.repo.UnitsForBuilding(ctx, buildingID)
		if err != nil {
			return err
		}
		known := map[uuid.UUID]bool{}
		for _, u := range units {
			known[u.ID] = true
		}
		for _, id := range in.UnitIDs {
			if !known[id] {
				return httpx.BadRequest("واحد انتخاب‌شده به این ساختمان تعلق ندارد")
			}
		}
		ci.TotalAmount = in.TotalAmount
		ci.FixedAmountPerUnit = nil
	case engine.MethodCombined:
		if in.TotalAmount <= 0 {
			return httpx.BadRequest(msgItemAmountInvalid)
		}
		if len(in.ComboWeights) == 0 {
			return httpx.BadRequest(engine.ErrBadWeights.Error())
		}
		sum := 0
		for _, w := range in.ComboWeights {
			if w.Weight <= 0 || w.Method == string(engine.MethodCombined) {
				return httpx.BadRequest(engine.ErrBadWeights.Error())
			}
			sum += w.Weight
		}
		if sum != 100 {
			return httpx.BadRequest(engine.ErrBadWeights.Error())
		}
		ci.TotalAmount = in.TotalAmount
		ci.FixedAmountPerUnit = nil
	default:
		return httpx.BadRequest(msgItemMethodInvalid)
	}
	return nil
}

// AddCostItem adds a cost item (draft periods only).
func (s *PeriodService) AddCostItem(ctx context.Context, manager *auth.User, periodID uuid.UUID, in CostItemInput) (*CostItem, error) {
	p, err := s.getAuthorizedPeriod(ctx, manager, periodID)
	if err != nil {
		return nil, err
	}
	if p.Status != PeriodDraft {
		return nil, httpx.Conflict("افزودن قلم هزینه فقط در وضعیت پیش‌نویس مجاز است")
	}
	ci := &CostItem{ID: uuid.New(), PeriodID: periodID}
	if err := s.validateCostItem(ctx, ci, &in, p.BuildingID); err != nil {
		return nil, err
	}
	if err := s.repo.CreateCostItem(ctx, ci); err != nil {
		return nil, err
	}
	return ci, nil
}

// authorizedCostItem loads a cost item through its period's building scope.
func (s *PeriodService) authorizedCostItem(ctx context.Context, manager *auth.User, id uuid.UUID) (*CostItem, *BillingPeriod, error) {
	ci, err := s.repo.GetCostItem(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, httpx.NotFound("قلم هزینه یافت نشد")
		}
		return nil, nil, err
	}
	p, err := s.repo.GetPeriod(ctx, ci.PeriodID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.authorizeBuilding(ctx, manager, p.BuildingID); err != nil {
		return nil, nil, err
	}
	return ci, p, nil
}

// UpdateCostItem edits a cost item (draft periods only).
func (s *PeriodService) UpdateCostItem(ctx context.Context, manager *auth.User, id uuid.UUID, in CostItemInput) (*CostItem, *CostItem, error) {
	ci, p, err := s.authorizedCostItem(ctx, manager, id)
	if err != nil {
		return nil, nil, err
	}
	if p.Status != PeriodDraft {
		return nil, nil, httpx.Conflict("ویرایش قلم هزینه فقط در وضعیت پیش‌نویس مجاز است")
	}
	before := *ci
	if err := s.validateCostItem(ctx, ci, &in, p.BuildingID); err != nil {
		return nil, nil, err
	}
	if err := s.repo.SaveCostItem(ctx, ci); err != nil {
		return nil, nil, err
	}
	return ci, &before, nil
}

// DeleteCostItem removes a draft cost item (draft periods only).
func (s *PeriodService) DeleteCostItem(ctx context.Context, manager *auth.User, id uuid.UUID) error {
	ci, p, err := s.authorizedCostItem(ctx, manager, id)
	if err != nil {
		return err
	}
	if p.Status != PeriodDraft {
		return httpx.Conflict("حذف قلم هزینه فقط در وضعیت پیش‌نویس مجاز است")
	}
	return s.repo.DeleteCostItem(ctx, ci.ID)
}

// --- state machine ----------------------------------------------------------------

// Reopen returns a calculated period to draft, discarding previews
// (calculated → draft). 409 when the period already issued.
func (s *PeriodService) Reopen(ctx context.Context, manager *auth.User, id uuid.UUID) (*BillingPeriod, error) {
	p, err := s.getAuthorizedPeriod(ctx, manager, id)
	if err != nil {
		return nil, err
	}
	if p.Status != PeriodCalculated {
		return nil, httpx.Conflict("بازگشایی فقط برای دوره محاسبه‌شده و پیش از صدور مجاز است")
	}
	if err := s.repo.DiscardCalculation(ctx, id); err != nil {
		return nil, err
	}
	p.Status = PeriodDraft
	if err := s.repo.SavePeriod(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Issue freezes the calculation and publishes every invoice: assigns
// sequential per-building numbers, stamps issued_at/inputs_frozen_at, emits
// invoice_issued notifications, and writes an audit entry per invoice
// (FR-038). Idempotent guard: only a calculated period can issue (409).
func (s *PeriodService) Issue(ctx context.Context, manager *auth.User, id uuid.UUID) ([]Invoice, error) {
	p, err := s.getAuthorizedPeriod(ctx, manager, id)
	if err != nil {
		return nil, err
	}
	if p.Status != PeriodCalculated {
		return nil, httpx.Conflict("صدور فقط برای دوره محاسبه‌شده مجاز است")
	}

	issued, err := s.repo.IssueInvoices(ctx, id, time.Now().Year(), time.Now())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound(msgPeriodNotFound)
		}
		if errors.Is(err, ErrNotCalculable) {
			return nil, httpx.Conflict("صدور فقط برای دوره محاسبه‌شده مجاز است")
		}
		return nil, err
	}

	refType := "invoice"
	// One notification per invoice to each active occupant of its unit
	// (in-app guaranteed channel, FR-032).
	for _, inv := range issued {
		uids, err := s.repo.OccupantUserIDs(ctx, []uuid.UUID{inv.UnitID})
		if err != nil {
			continue
		}
		for _, uid := range uids {
			_, _ = s.notif.Create(ctx, uid, "invoice_issued",
				fmt.Sprintf("صورتحساب %s صادر شد", p.Title),
				fmt.Sprintf("مبلغ قابل پرداخت: %d تومان", int64(inv.FinalAmount)),
				&refType, &inv.ID)
		}
	}

	// Audit each issued invoice (FR-038: invoice issue).
	actorID := manager.ID
	for i := range issued {
		inv := issued[i]
		_ = s.audit.Append(ctx, &actorID, "invoice.issued", "invoice", &inv.ID, nil, inv)
	}
	return issued, nil
}

// Close finishes an issued period (issued → closed).
func (s *PeriodService) Close(ctx context.Context, manager *auth.User, id uuid.UUID) (*BillingPeriod, error) {
	p, err := s.getAuthorizedPeriod(ctx, manager, id)
	if err != nil {
		return nil, err
	}
	if p.Status != PeriodIssued {
		return nil, httpx.Conflict("بستن دوره فقط پس از صدور مجاز است")
	}
	now := time.Now()
	p.Status = PeriodClosed
	p.ClosedAt = &now
	if err := s.repo.SavePeriod(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// --- invoice corrections -------------------------------------------------------------

// CancelInvoiceInput is the cancel payload (BR-10).
type CancelInvoiceInput struct {
	Reason string `json:"reason" binding:"required"`
}

// CancelInvoice cancels an invoice: the row and its history are preserved
// (BR-10); only unpaid/partial/expired invoices can be cancelled. Audited.
func (s *PeriodService) CancelInvoice(ctx context.Context, manager *auth.User, invoiceID uuid.UUID, in CancelInvoiceInput) (*Invoice, error) {
	inv, err := s.authorizedInvoice(ctx, manager, invoiceID)
	if err != nil {
		return nil, err
	}
	switch inv.Status {
	case InvoiceUnpaid, InvoicePartial, InvoiceExpired:
	default:
		return nil, httpx.Conflict(msgInvoiceNotCancellable)
	}
	if err := s.repo.SetInvoiceStatus(ctx, invoiceID, InvoiceCanceled); err != nil {
		return nil, err
	}
	inv.Status = InvoiceCanceled
	actorID := manager.ID
	_ = s.audit.Append(ctx, &actorID, "invoice.cancelled", "invoice", &invoiceID, nil,
		map[string]any{"reason": in.Reason, "invoice": inv})
	return inv, nil
}

// AdjustmentInput is the explicit-correction payload (FR-017).
type AdjustmentInput struct {
	Kind   string `json:"kind" binding:"required"`
	Amount Money  `json:"amount" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

// AddAdjustment appends a debit/credit note to a non-cancelled invoice. The
// original amounts are untouched; the note appears as a separate item row.
// Audited.
func (s *PeriodService) AddAdjustment(ctx context.Context, manager *auth.User, invoiceID uuid.UUID, in AdjustmentInput) (*InvoiceAdjustment, error) {
	inv, err := s.authorizedInvoice(ctx, manager, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.Status == InvoiceCanceled {
		return nil, httpx.Conflict("اصلاحیه برای صورتحساب ابطال‌شده مجاز نیست")
	}
	if in.Kind != AdjustmentDebit && in.Kind != AdjustmentCredit {
		return nil, httpx.BadRequest("نوع اصلاحیه باید بدهکار یا بستانکار باشد")
	}
	if in.Amount <= 0 {
		return nil, httpx.BadRequest(msgAdjustAmountInvalid)
	}
	if in.Reason == "" {
		return nil, httpx.BadRequest(msgAdjustReasonRequired)
	}
	adj := &InvoiceAdjustment{
		ID:        uuid.New(),
		InvoiceID: invoiceID,
		Kind:      in.Kind,
		Amount:    in.Amount,
		Reason:    in.Reason,
		CreatedBy: &manager.ID,
	}
	item := &InvoiceItem{
		ID:        uuid.New(),
		InvoiceID: invoiceID,
		Kind:      ItemAdjustment,
		Title:     adjustmentTitle(in.Kind, in.Reason),
		Amount:    in.Amount,
	}
	if err := s.repo.CreateAdjustment(ctx, adj, item); err != nil {
		return nil, err
	}
	actorID := manager.ID
	_ = s.audit.Append(ctx, &actorID, "invoice.adjusted", "invoice", &invoiceID, nil, adj)
	return adj, nil
}

// ListInvoices returns one filtered page of a permitted building's invoices.
func (s *PeriodService) ListInvoices(ctx context.Context, manager *auth.User, f InvoiceFilter) ([]Invoice, int64, error) {
	if err := s.authorizeBuilding(ctx, manager, f.BuildingID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListInvoices(ctx, f)
}

// authorizedInvoice loads an invoice and checks the manager scope of its
// building.
func (s *PeriodService) authorizedInvoice(ctx context.Context, manager *auth.User, invoiceID uuid.UUID) (*Invoice, error) {
	inv, err := s.repo.GetInvoice(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound(msgInvoiceNotFound)
		}
		return nil, err
	}
	if err := s.authorizeBuilding(ctx, manager, inv.BuildingID); err != nil {
		return nil, err
	}
	return inv, nil
}

// adjustmentTitle renders the adjustment item title.
func adjustmentTitle(kind, reason string) string {
	if kind == AdjustmentCredit {
		return "اصلاحیه بستانکار: " + reason
	}
	return "اصلاحیه بدهکار: " + reason
}
