package payment

// Service: manual recording (manager, audited via the handler middleware),
// gateway start/callback, the ledger queries, and unit-balance reads. Every
// payment application runs in one transaction that also recomputes the unit
// balance, so balances can never drift from the payment rows. Authz is
// object-level (FR-037): the invoice's building manager or an active
// resident of the unit.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/auth"
	"hamsa/internal/building"
	"hamsa/internal/notification"
	"hamsa/internal/payment/gateway"
	"hamsa/internal/platform/httpx"
)

// Persian service messages (contracts/api.md — all user-facing text is
// Persian).
const (
	msgInvoiceNotFound  = "صورتحساب یافت نشد"
	msgInvoiceCancelled = "صورتحساب لغو شده است؛ پرداخت ثبت نمی‌شود"
	msgAmountInvalid    = "مبلغ پرداخت باید بزرگ‌تر از صفر باشد"
	msgNotPermitted     = "شما به این بخش دسترسی ندارید"
	msgCallbackFailed   = "پرداخت ناموفق بود یا تأیید نشد"
	msgCallbackUnknown  = "پرداختی برای این بازگشت یافت نشد"
)

// Invoice status values mirrored from migration 0004 invoice_status.
const (
	invoiceUnpaid    = "unpaid"
	invoicePartial   = "partial"
	invoicePaid      = "paid"
	invoiceCancelled = "cancelled"
)

// PaymentService records and verifies payments.
type PaymentService struct {
	repo     *Repository
	balances *BalanceService
	gw       gateway.PaymentGateway
	notif    *notification.Service
	log      *slog.Logger
	callback string // absolute URL of the public GET /payments/callback
	now      func() time.Time
}

// NewPaymentService returns the payment service.
func NewPaymentService(repo *Repository, gw gateway.PaymentGateway, balances *BalanceService,
	notif *notification.Service, log *slog.Logger, callbackURL string) *PaymentService {
	return &PaymentService{
		repo:     repo,
		balances: balances,
		gw:       gw,
		notif:    notif,
		log:      log,
		callback: callbackURL,
		now:      time.Now,
	}
}

// ManualPaymentInput is the manager's record-payment payload (contracts/api.md).
// Amount arrives as a quoted integer string on the wire; json.Number also
// tolerates a raw number so Persian-digit client normalization never 400s.
type ManualPaymentInput struct {
	Amount         json.Number `json:"amount"`
	PaidAt         string      `json:"paid_at"`
	TrackingNumber string      `json:"tracking_number"`
}

// paymentApplication is the outcome of applying money to an invoice: how
// much cleared it and the new invoice status.
type paymentApplication struct {
	applied Money
	status  string
}

// applyToInvoice caps the payment at the invoice's effective outstanding,
// advances paid_amount, and derives the new invoice status. Caller holds the
// invoice row lock.
func applyToInvoice(inv *ledgerInvoice, amount Money) paymentApplication {
	eff := inv.effective()
	outstanding := eff - inv.PaidAmount
	if outstanding < 0 {
		outstanding = 0
	}
	applied := amount
	if applied > outstanding {
		applied = outstanding
	}
	newPaid := inv.PaidAmount + applied
	status := invoiceUnpaid
	switch {
	case newPaid >= eff:
		status = invoicePaid
	case newPaid > 0:
		status = invoicePartial
	}
	return paymentApplication{applied: applied, status: status}
}

// canAccessUnit reports whether the user may act on the unit: its building's
// manager or an active resident of the unit.
func (s *PaymentService) canAccessUnit(ctx context.Context, u *auth.User, unitID uuid.UUID) (bool, error) {
	unit, err := s.repo.UnitByID(ctx, unitID)
	if err != nil {
		return false, err
	}
	if u.Role == auth.RoleManager {
		return s.repo.isManagerOf(ctx, u.ID, unit.BuildingID)
	}
	units, err := s.repo.ResidentUnitIDs(ctx, u.Phone)
	if err != nil {
		return false, err
	}
	for _, id := range units {
		if id == unitID {
			return true, nil
		}
	}
	return false, nil
}

// authorizedInvoice loads the invoice and checks the caller's object-level
// access (manager of the building or resident of the unit).
func (s *PaymentService) authorizedInvoice(ctx context.Context, u *auth.User, invoiceID uuid.UUID) (*ledgerInvoice, error) {
	inv, err := s.repo.InvoiceForUpdate(ctx, nil, invoiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound(msgInvoiceNotFound)
		}
		return nil, err
	}
	ok, err := s.canAccessUnit(ctx, u, inv.UnitID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgNotPermitted)
	}
	return inv, nil
}

// RecordManual records a manager-confirmed payment against an issued
// invoice (FR-022). Surplus beyond the outstanding becomes unit credit.
func (s *PaymentService) RecordManual(ctx context.Context, u *auth.User, invoiceID uuid.UUID, in ManualPaymentInput) (*Payment, error) {
	if u.Role != auth.RoleManager {
		return nil, httpx.Forbidden(msgNotPermitted)
	}
	amount, err := strconv.ParseInt(in.Amount.String(), 10, 64)
	if err != nil || amount <= 0 {
		return nil, httpx.BadRequest(msgAmountInvalid)
	}
	var paidAt building.Date
	if in.PaidAt == "" {
		paidAt = building.Date{Time: s.now().UTC().Truncate(24 * time.Hour)}
	} else {
		t, err := time.Parse("2006-01-02", in.PaidAt)
		if err != nil {
			return nil, httpx.BadRequest("تاریخ پرداخت نامعتبر است")
		}
		paidAt = building.Date{Time: t}
	}

	inv, err := s.authorizedInvoice(ctx, u, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.Status == invoiceCancelled {
		return nil, httpx.Conflict(msgInvoiceCancelled)
	}

	p := &Payment{
		ID:         uuid.New(),
		BuildingID: inv.BuildingID,
		UnitID:     inv.UnitID,
		InvoiceID:  &invoiceID,
		Method:     MethodManual,
		Amount:     amount,
		PaidAt:     paidAt,
		Status:     StatusVerified, // manager-confirmed cash counts immediately
		RecordedBy: &u.ID,
	}
	if in.TrackingNumber != "" {
		p.TrackingNumber = &in.TrackingNumber
	}

	app := applyToInvoice(inv, amount)
	err = s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		if err := s.repo.ApplyInvoicePaid(ctx, tx, invoiceID, inv.PaidAmount+app.applied, app.status); err != nil {
			return err
		}
		_, err := s.balances.recomputeWithinTx(ctx, tx, inv.UnitID)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.notifyRecorded(ctx, p)
	return p, nil
}

// StartGateway opens an online payment for the invoice's outstanding (or the
// requested amount). The payment row starts `recorded` (pending verify); only
// the callback verify — always against the DB amount — completes it.
func (s *PaymentService) StartGateway(ctx context.Context, u *auth.User, invoiceID uuid.UUID, amount *Money) (*Payment, string, error) {
	inv, err := s.authorizedInvoice(ctx, u, invoiceID)
	if err != nil {
		return nil, "", err
	}
	if inv.Status == invoiceCancelled {
		return nil, "", httpx.Conflict(msgInvoiceCancelled)
	}
	outstanding := inv.effective() - inv.PaidAmount
	if outstanding < 0 {
		outstanding = 0
	}
	amt := outstanding
	if amount != nil {
		if *amount <= 0 {
			return nil, "", httpx.BadRequest(msgAmountInvalid)
		}
		amt = *amount
	}
	if amt <= 0 {
		return nil, "", httpx.Conflict("این صورتحساب بدهی معوق ندارد")
	}

	res, err := s.gw.Start(ctx, gateway.StartRequest{
		AmountToman: amt,
		CallbackURL: s.callback,
		Description: "پرداخت شارژ صورتحساب " + inv.InvoiceNumber,
	})
	if err != nil {
		s.log.Error("gateway start failed", "error", err)
		return nil, "", httpx.Internal("ارتباط با درگاه پرداخت ناموفق بود")
	}

	authority := res.Authority
	p := &Payment{
		ID:         uuid.New(),
		BuildingID: inv.BuildingID,
		UnitID:     inv.UnitID,
		InvoiceID:  &invoiceID,
		Method:     MethodGateway,
		Amount:     amt,
		PaidAt:     building.Date{Time: s.now().UTC().Truncate(24 * time.Hour)},
		Authority:  &authority,
		Gateway:    strPtr(s.gw.Name()),
		RecordedBy: &u.ID,
		Status:     StatusRecorded, // pending verify; not counted toward balances
	}
	if err := s.repo.CreatePayment(ctx, p); err != nil {
		return nil, "", err
	}
	return p, res.PayURL, nil
}

// HandleCallback resolves the payment BY OUR DB (payment_id for the mock
// flow, authority for real gateways), verifies against the DB amount, and
// finalizes. Both outcomes return 200 with the payment status (the gateway
// page renders it); already-finalized payments return their current state
// (idempotent gateway retries).
func (s *PaymentService) HandleCallback(ctx context.Context, paymentID, authority string) (*Payment, error) {
	var (
		p   *Payment
		err error
	)
	switch {
	case paymentID != "":
		id, parseErr := uuid.Parse(paymentID)
		if parseErr != nil {
			return nil, httpx.NotFound(msgCallbackUnknown)
		}
		p, err = s.repo.GetPayment(ctx, id)
	case authority != "":
		p, err = s.repo.GetPaymentByAuthority(ctx, authority)
	default:
		return nil, httpx.BadRequest("پارامتر payment_id یا Authority الزامی است")
	}
	if p == nil {
		return nil, httpx.NotFound(msgCallbackUnknown)
	}
	if p.Status != StatusRecorded || p.Method != MethodGateway || p.Authority == nil {
		return p, nil // already verified/failed — idempotent replay
	}

	// THE critical rule (research.md R5): the verified amount is the DB row's
	// amount, never anything the callback carried.
	res, verr := s.gw.Verify(ctx, *p.Authority, p.Amount)
	if verr != nil || !res.OK {
		s.log.Warn("gateway verify failed", "payment_id", p.ID, "error", verr)
		p.Status = StatusFailed
		if err := s.repo.SavePayment(ctx, p); err != nil {
			return nil, err
		}
		return p, nil // status=failed — the gateway page handles it
	}

	ref := res.RefID
	p.Status = StatusVerified
	p.TrackingNumber = &ref // the receipt reference replaces the authority
	err = s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(p).Error; err != nil {
			return err
		}
		if p.InvoiceID == nil {
			return nil
		}
		inv, err := s.repo.InvoiceForUpdate(ctx, tx, *p.InvoiceID)
		if err != nil {
			return err
		}
		app := applyToInvoice(inv, p.Amount)
		if err := s.repo.ApplyInvoicePaid(ctx, tx, inv.ID, inv.PaidAmount+app.applied, app.status); err != nil {
			return err
		}
		_, err = s.balances.recomputeWithinTx(ctx, tx, inv.UnitID)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.notifyRecorded(ctx, p)
	return p, nil
}

// UnitBalanceFor returns the unit's fresh balance row after an object-level
// access check; units with no financial history read as clean zeros.
func (s *PaymentService) UnitBalanceFor(ctx context.Context, u *auth.User, unitID uuid.UUID) (*UnitBalance, error) {
	ok, err := s.canAccessUnit(ctx, u, unitID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound("واحد یافت نشد")
		}
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgNotPermitted)
	}
	// Always fresh: read-your-writes even when a billing event (issue,
	// cancel, adjustment) has not triggered its hook yet.
	return s.balances.RecomputeUnit(ctx, unitID)
}

// BuildingLedger returns one filtered ledger page for a permitted manager.
func (s *PaymentService) BuildingLedger(ctx context.Context, u *auth.User, buildingID uuid.UUID, f PaymentFilter) ([]Payment, int64, error) {
	if u.Role != auth.RoleManager {
		return nil, 0, httpx.Forbidden(msgNotPermitted)
	}
	ok, err := s.repo.isManagerOf(ctx, u.ID, buildingID)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, httpx.Forbidden(msgNotPermitted)
	}
	f.BuildingID = buildingID
	return s.repo.ListPayments(ctx, f)
}

// MyPayments returns one page of the resident's payment history across the
// units they occupy.
func (s *PaymentService) MyPayments(ctx context.Context, u *auth.User, f PaymentFilter) ([]Payment, int64, error) {
	unitIDs, err := s.repo.ResidentUnitIDs(ctx, u.Phone)
	if err != nil {
		return nil, 0, err
	}
	if len(unitIDs) == 0 {
		return []Payment{}, 0, nil
	}
	f.UnitIDs = unitIDs
	return s.repo.ListPayments(ctx, f)
}

// notifyRecorded emits payment_recorded notifications to the unit's active
// occupants (best-effort — never fails the payment).
func (s *PaymentService) notifyRecorded(ctx context.Context, p *Payment) {
	users, err := s.repo.OccupantUserIDs(ctx, p.UnitID)
	if err != nil {
		s.log.Warn("payment notification lookup failed", "payment_id", p.ID, "error", err)
		return
	}
	typ := "payment_recorded"
	title := "پرداخت ثبت شد"
	body := fmt.Sprintf("پرداختی به مبلغ %s تومان برای واحد شما ثبت شد.", formatToman(p.Amount))
	ref := "payment"
	refID := p.ID
	for _, uid := range users {
		if _, err := s.notif.Create(ctx, uid, typ, title, body, &ref, &refID); err != nil {
			s.log.Warn("payment notification failed", "payment_id", p.ID, "user", uid, "error", err)
		}
	}
}

func strPtr(s string) *string { return &s }

// formatToman renders an amount with thousands separators (Persian digits
// are a client concern; the API stays numeric).
func formatToman(v Money) string {
	s := strconv.FormatInt(v, 10)
	if len(s) <= 3 {
		return s
	}
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	return out
}
