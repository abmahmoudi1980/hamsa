// Package engine is the pure charge-calculation core (US4/T044, research.md
// R7): six calculation methods, exact rational shares, largest-remainder
// rounding to integer Toman (BR-09), and late-fee rules (FR-016).
//
// ZERO infrastructure imports — no DB, HTTP, GORM, or config. Deterministic:
// rounding residue is distributed by largest fractional remainder with
// lowest-unit-id tie-break, so identical inputs always produce identical
// invoices (auditable, testable).
package engine

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
)

// Method is a cost-item calculation method (FR-011/FR-012, spec §7).
type Method string

const (
	MethodEqual         Method = "equal"          // A: split among participating units
	MethodPerOccupant   Method = "per_occupant"   // B: proportional to occupant count
	MethodPerArea       Method = "per_area"       // C: proportional to area m²
	MethodFixed         Method = "fixed"          // D: fixed amount per unit
	MethodSpecificUnits Method = "specific_units" // E: only listed units
	MethodCombined      Method = "combined"       // F: weighted mix of simple methods
)

// LateFeeType is a late-fee rule (FR-016).
type LateFeeType string

const (
	LateFeeNone    LateFeeType = "none"
	LateFeeFixed   LateFeeType = "fixed"   // value = Toman
	LateFeePercent LateFeeType = "percent" // value = % of base amount
	LateFeePerDay  LateFeeType = "per_day" // value = Toman per day late
)

// ComboWeight is one component of a combined method: the sub-method and its
// percentage weight. Weights must sum to 100.
type ComboWeight struct {
	Method             Method
	Weight             int
	FixedAmountPerUnit int64 // only meaningful when Method is fixed
}

// CostItem is the calculation input for one cost line.
type CostItem struct {
	ID                 string
	Total              int64 // Toman; for fixed the per-unit amount drives shares
	Method             Method
	FixedAmountPerUnit int64 // for fixed
	ComboWeights       []ComboWeight
	IncludeVacant      bool     // BR-06/BR-07 vacant-unit inclusion rule
	UnitIDs            []string // specific_units participants (explicit selection wins over vacancy)
}

// Unit is one unit's calculation inputs as snapshotted at calculation time.
type Unit struct {
	ID        string // uuid; lexicographic order is the rounding tie-break (R7)
	Occupants int
	AreaM2    int64
	Vacant    bool
}

// Share is one unit's share of one cost item.
type Share struct {
	UnitID  string
	Exact   *big.Rat // exact rational share, kept for audit (NUMERIC(20,4))
	Rounded int64    // after largest-remainder rounding
}

// ItemResult is the calculation output for one cost item.
type ItemResult struct {
	CostItemID string
	TotalUsed  int64 // the reconciliation target actually distributed
	Shares     []Share
	Reconciled bool // Σ Rounded == TotalUsed (BR-09) — surfaced for BR-08 preview
}

// Engine errors — all map to Persian messages at the service layer.
var (
	ErrNoParticipants     = errors.New("هیچ واحد مشارکت‌کننده‌ای برای این قلم هزینه وجود ندارد")
	ErrZeroFactor         = errors.New("عامل مشارکت صفر است (جمع شمارنده‌ها/متراژها صفر است)")
	ErrSpecificUnitsEmpty = errors.New("هیچ واحدی برای این قلم هزینه انتخاب نشده است")
	ErrBadWeights         = errors.New("وزن‌های روش ترکیبی نامعتبر است (باید جمعاً ۱۰۰ و بزرگ‌تر از صفر باشند)")
	ErrNonPositiveTotal   = errors.New("مبلغ قلم هزینه باید بزرگ‌تر از صفر باشد")
	ErrUnknownMethod      = errors.New("روش محاسبه نامعتبر است")
	ErrUnknownLateFee     = errors.New("نوع دیرکرد نامعتبر است")
)

// Calculate computes one cost item's shares across units.
func Calculate(item CostItem, units []Unit) (*ItemResult, error) {
	if item.Total <= 0 && item.Method != MethodFixed {
		return nil, fmt.Errorf("%w: قلم %s", ErrNonPositiveTotal, item.ID)
	}
	if item.Method == MethodFixed && item.FixedAmountPerUnit <= 0 {
		return nil, fmt.Errorf("%w: قلم %s", ErrNonPositiveTotal, item.ID)
	}

	switch item.Method {
	case MethodEqual:
		return distribute(item.ID, item.Total, participants(item, units), func(Unit) *big.Rat { return ratOne })
	case MethodPerOccupant:
		return distribute(item.ID, item.Total, participants(item, units), func(u Unit) *big.Rat { return new(big.Rat).SetInt64(int64(u.Occupants)) })
	case MethodPerArea:
		return distribute(item.ID, item.Total, participants(item, units), func(u Unit) *big.Rat { return new(big.Rat).SetInt64(u.AreaM2) })
	case MethodFixed:
		return calcFixed(item, units)
	case MethodSpecificUnits:
		return calcSpecific(item, units)
	case MethodCombined:
		return calcCombined(item, units)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownMethod, item.Method)
	}
}

// participants applies the vacant-inclusion rule (BR-06/BR-07). Explicit
// specific_units selection bypasses it — handled in calcSpecific.
func participants(item CostItem, units []Unit) []Unit {
	out := make([]Unit, 0, len(units))
	for _, u := range units {
		if u.Vacant && !item.IncludeVacant {
			continue
		}
		out = append(out, u)
	}
	return out
}

func calcFixed(item CostItem, units []Unit) (*ItemResult, error) {
	els := participants(item, units)
	if len(els) == 0 {
		return nil, ErrNoParticipants
	}
	// Reconciliation target for fixed is Σ (per-unit amount), i.e. the total
	// actually charged — the manager's nominal item total is informational.
	total := item.FixedAmountPerUnit * int64(len(els))
	shares := make([]Share, 0, len(els))
	for _, u := range els {
		shares = append(shares, Share{UnitID: u.ID, Exact: new(big.Rat).SetInt64(item.FixedAmountPerUnit), Rounded: item.FixedAmountPerUnit})
	}
	sortShares(shares)
	return &ItemResult{CostItemID: item.ID, TotalUsed: total, Shares: shares, Reconciled: true}, nil
}

func calcSpecific(item CostItem, units []Unit) (*ItemResult, error) {
	if len(item.UnitIDs) == 0 {
		return nil, ErrSpecificUnitsEmpty
	}
	want := make(map[string]bool, len(item.UnitIDs))
	for _, id := range item.UnitIDs {
		want[id] = true
	}
	els := make([]Unit, 0, len(item.UnitIDs))
	for _, u := range units {
		if want[u.ID] {
			els = append(els, u)
		}
	}
	if len(els) == 0 {
		return nil, ErrSpecificUnitsEmpty
	}
	return distribute(item.ID, item.Total, els, func(Unit) *big.Rat { return ratOne })
}

func calcCombined(item CostItem, units []Unit) (*ItemResult, error) {
	if len(item.ComboWeights) == 0 {
		return nil, ErrBadWeights
	}
	sum := 0
	for _, w := range item.ComboWeights {
		if w.Weight <= 0 || w.Method == MethodCombined {
			return nil, ErrBadWeights
		}
		sum += w.Weight
	}
	if sum != 100 {
		return nil, ErrBadWeights
	}

	// Allocate the total into per-method pools with largest remainder so the
	// pools themselves reconcile to the item total (BR-09 holds per sub-pool,
	// therefore for the combined item).
	exactPools := make([]*big.Rat, len(item.ComboWeights))
	for i, w := range item.ComboWeights {
		exactPools[i] = new(big.Rat).Quo(
			new(big.Rat).Mul(new(big.Rat).SetInt64(item.Total), new(big.Rat).SetInt64(int64(w.Weight))),
			new(big.Rat).SetInt64(100))
	}
	pools := allocateLargestRemainder(item.Total, exactPools)

	result := &ItemResult{CostItemID: item.ID, TotalUsed: item.Total, Shares: []Share{}, Reconciled: true}
	perUnit := make(map[string]*Share)
	var running int64
	for i, w := range item.ComboWeights {
		sub := CostItem{ID: item.ID, Total: pools[i], Method: w.Method,
			FixedAmountPerUnit: w.FixedAmountPerUnit, IncludeVacant: item.IncludeVacant}
		var res *ItemResult
		var err error
		// Fixed sub-method uses the weighted pool as an exact per-unit split
		// target: amount = pool / participants (fixed per-unit inside combos
		// has no separate per-unit amount).
		switch w.Method {
		case MethodFixed:
			res, err = distribute(sub.ID, pools[i], participants(sub, units), func(Unit) *big.Rat { return ratOne })
		default:
			res, err = Calculate(sub, units)
		}
		if err != nil {
			return nil, err
		}
		running += sumShares(res)
		for _, sh := range res.Shares {
			acc, ok := perUnit[sh.UnitID]
			if !ok {
				acc = &Share{UnitID: sh.UnitID, Exact: new(big.Rat)}
				perUnit[sh.UnitID] = acc
			}
			acc.Exact.Add(acc.Exact, sh.Exact)
			acc.Rounded += sh.Rounded
		}
	}
	for _, sh := range perUnit {
		result.Shares = append(result.Shares, *sh)
	}
	sortShares(result.Shares)
	result.Reconciled = running == item.Total
	return result, nil
}

var ratOne = new(big.Rat).SetInt64(1)

// distribute splits total across els proportionally to weight(u), using
// largest-remainder rounding (research.md R7). Σ weights must be > 0.
func distribute(itemID string, total int64, els []Unit, weight func(Unit) *big.Rat) (*ItemResult, error) {
	if len(els) == 0 {
		return nil, ErrNoParticipants
	}
	weights := make([]*big.Rat, len(els))
	var sum *big.Rat = new(big.Rat)
	for i, u := range els {
		weights[i] = weight(u)
		sum.Add(sum, weights[i])
	}
	if sum.Sign() == 0 {
		return nil, ErrZeroFactor
	}
	exacts := make([]*big.Rat, len(els))
	for i := range els {
		exacts[i] = new(big.Rat).Mul(new(big.Rat).SetInt64(total), new(big.Rat).Quo(weights[i], sum))
	}
	rounded := allocateLargestRemainder(total, exacts)
	shares := make([]Share, len(els))
	for i, u := range els {
		shares[i] = Share{UnitID: u.ID, Exact: exacts[i], Rounded: rounded[i]}
	}
	sortShares(shares)
	var roundedSum int64
	for _, sh := range shares {
		roundedSum += sh.Rounded
	}
	return &ItemResult{CostItemID: itemID, TotalUsed: total, Shares: shares, Reconciled: roundedSum == total}, nil
}

// allocateLargestRemainder rounds exact[i] (Σ exact == total) to integers so
// Σ rounded == total: floor each, then hand the remaining Toman units one by
// one to the largest fractional remainders; ties go to the lowest index
// (caller pre-sorts by unit id so this is the lowest-unit-id tie-break, R7).
func allocateLargestRemainder(total int64, exact []*big.Rat) []int64 {
	n := len(exact)
	out := make([]int64, n)
	type rem struct {
		idx  int
		frac *big.Rat
	}
	rems := make([]rem, 0, n)
	var floored int64
	for i, e := range exact {
		f := new(big.Int).Quo(e.Num(), e.Denom()) // floor for positive values
		out[i] = f.Int64()
		floored += out[i]
		frac := new(big.Rat).Sub(e, new(big.Rat).SetInt(f))
		rems = append(rems, rem{idx: i, frac: frac})
	}
	leftover := total - floored
	if leftover <= 0 {
		return out
	}
	// Descending fractional remainder; stable tie-break on ascending index.
	sort.SliceStable(rems, func(a, b int) bool {
		if c := rems[a].frac.Cmp(rems[b].frac); c != 0 {
			return c > 0
		}

		return rems[a].idx < rems[b].idx
	})
	for i := 0; i < int(leftover); i++ {
		out[rems[i].idx]++
	}
	return out
}
func sumShares(res *ItemResult) int64 {
	var s int64
	for _, sh := range res.Shares {
		s += sh.Rounded
	}
	return s
}

func sortShares(shares []Share) {
	sort.Slice(shares, func(a, b int) bool { return shares[a].UnitID < shares[b].UnitID })
}

// LateFee computes the late-fee amount for an invoice (FR-016): none, fixed
// Toman, percentage of the base amount, or Toman per day late. daysLate is
// supplied by the caller (calendar stays outside the pure engine); negative
// values are clamped to 0.
func LateFee(kind LateFeeType, value float64, baseAmount int64, daysLate int) (int64, error) {
	if daysLate < 0 {
		daysLate = 0
	}
	switch kind {
	case LateFeeNone:
		return 0, nil
	case LateFeeFixed:
		return roundRat(new(big.Rat).SetFloat64(value)), nil
	case LateFeePercent:
		exact := new(big.Rat).Mul(new(big.Rat).SetInt64(baseAmount), new(big.Rat).SetFloat64(value/100))
		return roundRat(exact), nil
	case LateFeePerDay:
		return int64(daysLate) * roundRat(new(big.Rat).SetFloat64(value)), nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnknownLateFee, kind)
	}
}

// roundRat rounds half away from zero (money-grade rounding for fee inputs).
func roundRat(r *big.Rat) int64 {
	f := new(big.Float).SetRat(r)
	i, _ := f.Int(nil)
	// .Int truncates toward zero; add 1 when the fraction ≥ 0.5.
	diff := new(big.Rat).Sub(r, new(big.Rat).SetInt(i))
	half := new(big.Rat).SetFloat64(0.5)
	if diff.Cmp(half) >= 0 {
		i.Add(i, big.NewInt(1))
	}
	return i.Int64()
}
