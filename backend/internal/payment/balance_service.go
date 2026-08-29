package payment

// T057 — the balance service: unit_balances rows recomputed transactionally
// on every payment/adjustment/invoice event (BR-01). The recompute is fully
// derived from invoices/adjustments/payments, so it is idempotent and cannot
// drift; every event handler simply calls it after mutating the source rows.
// Payments are never hard-deleted (BR-02) — reversal is a status change
// followed by a recompute.

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BalanceService owns unit_balances.
type BalanceService struct {
	repo *Repository
}

// NewBalanceService returns the balance service.
func NewBalanceService(repo *Repository) *BalanceService { return &BalanceService{repo: repo} }

// Snapshot returns (priorDebt, credit) for the charge engine's calculation
// snapshot (FR-015): the unit's outstanding balance becomes the new invoice's
// prior_debt and the credit asset becomes its credit_amount.
func (s *BalanceService) Snapshot(ctx context.Context, unitID uuid.UUID) (priorDebt, credit int64, err error) {
	ub, err := s.GetUnitBalance(ctx, unitID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, 0, nil // no financial history yet — clean slate
		}
		return 0, 0, err
	}
	return ub.Balance, ub.CreditAsset, nil
}

// GetUnitBalance returns the stored row (gorm.ErrRecordNotFound when the
// unit has no financial history yet).
func (s *BalanceService) GetUnitBalance(ctx context.Context, unitID uuid.UUID) (*UnitBalance, error) {
	return s.repo.GetUnitBalance(ctx, unitID)
}

// recomputeWithinTx recomputes a unit's balance inside the caller's
// transaction (payment application must update invoice + balance atomically).
func (s *BalanceService) recomputeWithinTx(ctx context.Context, tx *gorm.DB, unitID uuid.UUID) (*UnitBalance, error) {
	return s.repo.recomputeBalance(ctx, tx, unitID)
}

// RecomputeUnit recomputes one unit's balance on its own (payment recorded /
// reversed events).
func (s *BalanceService) RecomputeUnit(ctx context.Context, unitID uuid.UUID) (*UnitBalance, error) {
	return s.repo.recomputeBalance(ctx, nil, unitID)
}

func (s *BalanceService) RecomputeInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	var unitID string
	err := s.repo.db.WithContext(ctx).
		Raw(`SELECT unit_id FROM invoices WHERE id = ?`, invoiceID).Scan(&unitID).Error
	if err != nil {
		return err
	}
	id, err := uuid.Parse(unitID)
	if err != nil || id == uuid.Nil {
		return nil // invoice vanished between events — nothing to refresh
	}
	_, err = s.repo.recomputeBalance(ctx, nil, id)
	return err
}

// ReversePayment flips a payment to 'reversed' (compensating state — the row
// is never deleted) and refreshes the unit balance. Any invoice-level
// compensating adjustment is a manager flow, out of MVP automation scope.
func (s *BalanceService) ReversePayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error) {
	var out *Payment
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Payment
		if err := tx.Where("id = ?", paymentID).First(&p).Error; err != nil {
			return err
		}
		p.Status = StatusReversed
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		if _, err := s.recomputeWithinTx(ctx, tx, p.UnitID); err != nil {
			return err
		}
		out = &p
		return nil
	})
	return out, err
}
