package billing

// T047 — calculation service: snapshots engine inputs at calculation time
// (occupant counts as-of the period end date, areas, vacancy, balances),
// runs the pure engine, and persists cost_item_shares + draft invoices with
// prior debt, late fee, credit, and final amount (FR-015).

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/auth"
	"hamsa/internal/billing/engine"
	"hamsa/internal/building"
	"hamsa/internal/platform/httpx"
)

// BalanceProvider supplies the unit's financial position at calculation time
// (data-model.md: prior_debt/credit snapshot into the invoice). Phase 7
// (T057) replaces the zero provider with the transactional balance service.
type BalanceProvider interface {
	Snapshot(ctx context.Context, unitID uuid.UUID) (priorDebt, credit Money, err error)
}

// ZeroBalanceProvider is the Phase-6 default: no payments exist yet, so every
// unit starts from a clean slate.
type ZeroBalanceProvider struct{}

// Snapshot returns zero prior debt and zero credit.
func (ZeroBalanceProvider) Snapshot(context.Context, uuid.UUID) (Money, Money, error) {
	return 0, 0, nil
}

// CalcService runs and previews calculations.
type CalcService struct {
	repo     *Repository
	balances BalanceProvider
}

// NewCalcService returns the calculation service.
func NewCalcService(repo *Repository, balances BalanceProvider) *CalcService {
	if balances == nil {
		balances = ZeroBalanceProvider{}
	}
	return &CalcService{repo: repo, balances: balances}
}

// engineError maps pure-engine failures onto the API envelope (400 with the
// engine's Persian message).
func engineError(err error) error {
	var msg string
	switch {
	case errors.Is(err, engine.ErrNoParticipants),
		errors.Is(err, engine.ErrZeroFactor),
		errors.Is(err, engine.ErrSpecificUnitsEmpty),
		errors.Is(err, engine.ErrBadWeights),
		errors.Is(err, engine.ErrNonPositiveTotal),
		errors.Is(err, engine.ErrUnknownMethod),
		errors.Is(err, engine.ErrUnknownLateFee):
		msg = err.Error()
		// Engine messages carry an item suffix ("قلم X") when present; the
		// base message is already Persian and user-facing.
		return httpx.BadRequest(strings.SplitN(msg, ":", 2)[0])
	default:
		return err
	}
}

// buildingUnit is one unit's snapshot for the engine.
type buildingUnit struct {
	unit      building.Unit
	occupants int
}

// Calculate runs the charge engine for the period (draft or calculated —
// recalculation refreshes drafts) and persists the result. Returns the
// preview payload.
func (s *CalcService) Calculate(ctx context.Context, manager *auth.User, periodID uuid.UUID) (*Preview, error) {
	period, err := s.authorizedPeriod(ctx, manager, periodID)
	if err != nil {
		return nil, err
	}
	if period.Status != PeriodDraft && period.Status != PeriodCalculated {
		return nil, httpx.Conflict("محاسبه فقط برای دوره در وضعیت پیش‌نویس یا محاسبه‌شده مجاز است")
	}

	items, err := s.repo.ListCostItems(ctx, periodID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, httpx.BadRequest("هیچ قلم هزینه‌ای برای این دوره تعریف نشده است")
	}

	// chargeLine is one cost item's rounded share awaiting invoice composition.
	type chargeLine struct {
		title  string
		itemID uuid.UUID
		method string
		amount Money
	}

	units, err := s.repo.UnitsForBuilding(ctx, period.BuildingID)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, httpx.BadRequest("این ساختمان واحدی ندارد")
	}

	// Snapshot inputs as-of the period end date (data-model.md: the engine
	// reads the as-of occupant count, snapshotted at calculation).
	bus := make([]buildingUnit, len(units))
	engineUnits := make([]engine.Unit, len(units))
	for i, u := range units {
		occ, err := s.repo.OccupantCountAsOf(ctx, u.ID, period.EndDate.Time)
		if err != nil {
			return nil, err
		}
		bus[i] = buildingUnit{unit: u, occupants: occ}
		engineUnits[i] = engine.Unit{
			ID:        u.ID.String(),
			Occupants: occ,
			AreaM2:    int64(u.AreaM2),
			Vacant:    u.Status == building.UnitStatusVacant,
		}
	}

	shares := []CostItemShare{}
	invoices := []InvoiceWithItems{}
	chargeByUnit := map[uuid.UUID][]chargeLine{}
	for _, item := range items {
		ei, err := s.engineItem(item)
		if err != nil {
			return nil, err
		}
		res, err := engine.Calculate(ei, engineUnits)
		if err != nil {
			return nil, engineError(err)
		}
		itemID := item.ID
		method := string(item.Method)
		for _, sh := range res.Shares {
			unitID, err := uuid.Parse(sh.UnitID)
			if err != nil {
				return nil, err
			}
			snap := map[string]any{}
			for _, bu := range bus {
				if bu.unit.ID == unitID {
					snap = map[string]any{
						"occupants": bu.occupants,
						"area_m2":   bu.unit.AreaM2,
						"vacant":    bu.unit.Status == building.UnitStatusVacant,
					}
				}
			}
			shares = append(shares, CostItemShare{
				ID:             uuid.New(),
				CostItemID:     itemID,
				UnitID:         unitID,
				ExactShare:     exactShareString(sh.Exact),
				RoundedShare:   Money(sh.Rounded),
				InputsSnapshot: snap,
			})
		}

		// Accumulate the per-unit share rows; invoices are composed once per
		// unit after all items are processed (FR-015).
		for _, sh := range res.Shares {
			unitID, _ := uuid.Parse(sh.UnitID)
			chargeByUnit[unitID] = append(chargeByUnit[unitID], chargeLine{
				title:  item.Title,
				itemID: itemID,
				method: method,
				amount: Money(sh.Rounded),
			})
		}
	}

	// One invoice per unit (spec §11): base = Σ charge items, plus prior debt,
	// late fee, and credit as separate components/items.
	for _, bu := range bus {
		var base int64
		items := []InvoiceItem{}
		for _, line := range chargeByUnit[bu.unit.ID] {
			base += int64(line.amount)
			itemID := line.itemID
			method := line.method
			items = append(items, InvoiceItem{
				ID:         uuid.New(),
				Kind:       ItemCharge,
				Title:      line.title,
				CostItemID: &itemID,
				Amount:     line.amount,
				Method:     &method,
			})
		}
		priorDebt, credit, err := s.balances.Snapshot(ctx, bu.unit.ID)
		if err != nil {
			return nil, err
		}
		lateFee, err := engine.LateFee(
			engineLateFeeType(period.LateFeeType),
			period.LateFeeValue,
			base,
			0, // days late at calculation: due date is in the future
		)
		if err != nil {
			return nil, engineError(err)
		}
		final := base + int64(priorDebt) + lateFee - int64(credit)

		inv := Invoice{
			ID:            uuid.New(),
			InvoiceNumber: "DRAFT-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24],
			BuildingID:    period.BuildingID,
			PeriodID:      period.ID,
			UnitID:        bu.unit.ID,
			BaseAmount:    Money(base),
			PriorDebt:     priorDebt,
			LateFeeAmount: Money(lateFee),
			CreditAmount:  credit,
			FinalAmount:   Money(final),
			DueDate:       &period.DueDate,
			Status:        InvoiceUnpaid,
		}
		if lateFee > 0 {
			lf := period.LateFeeType
			items = append(items, InvoiceItem{
				ID:     uuid.New(),
				Kind:   ItemLateFee,
				Title:  lateFeeTitle(period.LateFeeType),
				Amount: Money(lateFee),
				Method: &lf,
			})
		}
		invoices = append(invoices, InvoiceWithItems{Invoice: inv, Items: items})
	}

	if err := s.repo.ReplaceCalculation(ctx, periodID, shares, invoices); err != nil {
		return nil, err
	}
	period.Status = PeriodCalculated
	if err := s.repo.SavePeriod(ctx, period); err != nil {
		return nil, err
	}
	return s.Preview(ctx, manager, periodID)
}

// Preview returns the reviewable breakdown (BR-08, FR-014): per cost item —
// method, total, each unit's exact + rounded share; per unit — the resulting
// invoice amounts, with the reconciliation flag (BR-09).
func (s *CalcService) Preview(ctx context.Context, manager *auth.User, periodID uuid.UUID) (*Preview, error) {
	period, err := s.authorizedPeriod(ctx, manager, periodID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListCostItems(ctx, periodID)
	if err != nil {
		return nil, err
	}
	shares, err := s.repo.ListSharesForPeriod(ctx, periodID)
	if err != nil {
		return nil, err
	}
	invFilter := InvoiceFilter{BuildingID: period.BuildingID, PeriodID: &periodID, Page: 1, PageSize: 100}
	invoices, _, err := s.repo.ListInvoices(ctx, invFilter)
	if err != nil {
		return nil, err
	}
	units, err := s.repo.UnitsForBuilding(ctx, period.BuildingID)
	if err != nil {
		return nil, err
	}
	unitNumber := map[uuid.UUID]string{}
	for _, u := range units {
		unitNumber[u.ID] = u.Number
	}

	byItem := map[uuid.UUID][]CostItemShare{}
	for _, sh := range shares {
		byItem[sh.CostItemID] = append(byItem[sh.CostItemID], sh)
	}

	preview := &Preview{
		Period:     period,
		Items:      []PreviewItem{},
		Reconciled: true,
	}
	for _, item := range items {
		its := byItem[item.ID]
		pi := PreviewItem{
			CostItem:   item,
			Shares:     []PreviewShare{},
			Reconciled: true,
		}
		var sum Money
		for _, sh := range its {
			sum += sh.RoundedShare
			pi.Shares = append(pi.Shares, PreviewShare{
				UnitID:       sh.UnitID,
				UnitNumber:   unitNumber[sh.UnitID],
				ExactShare:   sh.ExactShare,
				RoundedShare: sh.RoundedShare,
				Inputs:       sh.InputsSnapshot,
			})
		}
		// Fixed items reconcile against Σ (per-unit amount), not the nominal
		// total (see engine calcFixed).
		target := int64(item.TotalAmount)
		if item.Method == "fixed" && item.FixedAmountPerUnit != nil {
			n := len(pi.Shares)
			if item.IncludeVacant {
				// participants = all live units is not known here exactly;
				// use the share rows themselves.
				n = len(its)
			}
			target = int64(*item.FixedAmountPerUnit) * int64(n)
		}
		pi.TotalUsed = Money(target)
		pi.Reconciled = int64(sum) == target
		if !pi.Reconciled {
			preview.Reconciled = false
		}
		preview.Items = append(preview.Items, pi)
	}

	preview.Invoices = make([]Invoice, 0, len(invoices))
	for i := range invoices {
		inv := invoices[i]
		inv.Items = nil
		preview.Invoices = append(preview.Invoices, inv)
	}
	return preview, nil
}

// engineItem converts a stored cost item into the pure-engine input.
func (s *CalcService) engineItem(item CostItem) (engine.CostItem, error) {
	ei := engine.CostItem{
		ID:            item.ID.String(),
		Total:         int64(item.TotalAmount),
		Method:        engine.Method(item.Method),
		IncludeVacant: item.IncludeVacant,
	}
	if item.FixedAmountPerUnit != nil {
		ei.FixedAmountPerUnit = int64(*item.FixedAmountPerUnit)
	}
	for _, id := range item.UnitIDs {
		ei.UnitIDs = append(ei.UnitIDs, id.String())
	}
	for _, w := range item.ComboWeights {
		ei.ComboWeights = append(ei.ComboWeights, engine.ComboWeight{
			Method:             engine.Method(w.Method),
			Weight:             w.Weight,
			FixedAmountPerUnit: int64(w.FixedAmountPerUnit),
		})
	}
	return ei, nil
}

// authorizedPeriod loads a period and verifies the manager scope (403
// regardless of existence when not permitted — contracts/api.md).
func (s *CalcService) authorizedPeriod(ctx context.Context, manager *auth.User, id uuid.UUID) (*BillingPeriod, error) {
	p, err := s.repo.GetPeriod(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound("دوره صدور صورتحساب یافت نشد")
		}
		return nil, err
	}
	ok, err := s.repo.IsManagerOf(ctx, manager.ID, p.BuildingID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden("شما به این ساختمان دسترسی ندارید")
	}
	return p, nil
}

// lateFeeTitle renders the late-fee invoice item title (FR-016).
func lateFeeTitle(kind string) string {
	switch kind {
	case "fixed":
		return "دیرکرد (مبلغ ثابت)"
	case "percent":
		return "دیرکرد (درصدی)"
	case "per_day":
		return "دیرکرد (روزانه)"
	default:
		return "دیرکرد"
	}
}

// --- preview payload -----------------------------------------------------------

// PreviewShare is one unit's share of one cost item in the preview.
type PreviewShare struct {
	UnitID       uuid.UUID      `json:"unit_id"`
	UnitNumber   string         `json:"unit_number,omitempty"`
	ExactShare   string         `json:"exact_share"`
	RoundedShare Money          `json:"rounded_share"`
	Inputs       map[string]any `json:"inputs_snapshot"`
}

// PreviewItem is one cost item's reviewable breakdown.
type PreviewItem struct {
	CostItem
	TotalUsed  Money          `json:"total_used"`
	Shares     []PreviewShare `json:"shares"`
	Reconciled bool           `json:"reconciled"`
}

// Preview is the GET /periods/{id}/preview response (BR-08).
type Preview struct {
	Period     *BillingPeriod `json:"period"`
	Items      []PreviewItem  `json:"items"`
	Invoices   []Invoice      `json:"invoices"`
	Reconciled bool           `json:"reconciled"`
}
