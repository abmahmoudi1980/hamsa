package engine

// T042 — pure table-driven charge-engine tests (US4). Covers all six
// calculation methods, the spec §25 acceptance scenario (unit 1 = 2,600,000),
// largest-remainder reconciliation (Σ shares ≡ total, BR-09), combined
// weights, zero-participating-factor errors, and vacant-unit inclusion rules
// (BR-06/BR-07). No infrastructure imports — the engine under test is pure.

import (
	"errors"
	"math/big"
	"testing"
)

// tenUnits builds the spec §25 fixture: 10 units, unit "u1" has 4 occupants
// and the rest 16 occupants spread so Σ = 20. All areas equal.
func tenUnits() []Unit {
	units := make([]Unit, 0, 10)
	for i := 1; i <= 10; i++ {
		id := "u" + itoa(i)
		occ := 0
		switch i {
		case 1:
			occ = 4
		case 2:
			occ = 6
		case 3:
			occ = 5
		case 4:
			occ = 5
		}
		units = append(units, Unit{ID: id, Occupants: occ, AreaM2: 100})
	}
	return units
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

func shareOf(res *ItemResult, unitID string) *Share {
	for i := range res.Shares {
		if res.Shares[i].UnitID == unitID {
			return &res.Shares[i]
		}
	}
	return nil
}

func sumRounded(res *ItemResult) int64 {
	var s int64
	for _, sh := range res.Shares {
		s += sh.Rounded
	}
	return s
}

// --- spec §25 acceptance ------------------------------------------------------

// Spec §25 / quickstart Scenario 2: 10 units, 10,000,000 equal + 8,000,000
// per-occupant (20 occupants, unit 1 = 4) → unit 1 invoice = 2,600,000.
func TestSpec25Scenario(t *testing.T) {
	units := tenUnits()

	equalRes, err := Calculate(CostItem{ID: "e", Total: 10_000_000, Method: MethodEqual}, units)
	if err != nil {
		t.Fatalf("equal: %v", err)
	}
	occRes, err := Calculate(CostItem{ID: "o", Total: 8_000_000, Method: MethodPerOccupant}, units)
	if err != nil {
		t.Fatalf("per_occupant: %v", err)
	}

	if got := shareOf(equalRes, "u1").Rounded; got != 1_000_000 {
		t.Fatalf("unit1 equal share = %d, want 1,000,000", got)
	}
	if got := shareOf(occRes, "u1").Rounded; got != 1_600_000 {
		t.Fatalf("unit1 per-occupant share = %d, want 1,600,000", got)
	}
	if total := shareOf(equalRes, "u1").Rounded + shareOf(occRes, "u1").Rounded; total != 2_600_000 {
		t.Fatalf("unit1 invoice = %d, want 2,600,000 (spec §25)", total)
	}

	// BR-09 reconciliation on both items.
	if !equalRes.Reconciled || sumRounded(equalRes) != 10_000_000 {
		t.Fatalf("equal item not reconciled: Σ=%d", sumRounded(equalRes))
	}
	if !occRes.Reconciled || sumRounded(occRes) != 8_000_000 {
		t.Fatalf("per-occupant item not reconciled: Σ=%d", sumRounded(occRes))
	}
	// Every other unit: 1,000,000 + 8,000,000×occ/20.
	for i := 2; i <= 10; i++ {
		id := "u" + itoa(i)
		occ := map[string]int{"u2": 6, "u3": 5, "u4": 5}[id]
		want := int64(1_000_000) + int64(occ)*400_000
		if got := shareOf(equalRes, id).Rounded + shareOf(occRes, id).Rounded; got != want {
			t.Fatalf("%s invoice = %d, want %d", id, got, want)
		}
	}
}

// --- largest-remainder reconciliation (BR-09) ---------------------------------

func TestLargestRemainderReconciliation(t *testing.T) {
	units := []Unit{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	// 10,000,001 is not divisible by 3: floors are 3,333,333 each, remainder
	// 2 Toman goes to the largest fractional remainders — all equal (1/3), so
	// the tie-break picks the lowest unit ids: "a" and "b".
	res, err := Calculate(CostItem{ID: "x", Total: 10_000_001, Method: MethodEqual}, units)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if sumRounded(res) != 10_000_001 {
		t.Fatalf("Σ rounded = %d, want 10,000,001", sumRounded(res))
	}
	if shareOf(res, "a").Rounded != 3_333_334 || shareOf(res, "b").Rounded != 3_333_334 ||
		shareOf(res, "c").Rounded != 3_333_333 {
		t.Fatalf("tie-break wrong: a=%d b=%d c=%d",
			shareOf(res, "a").Rounded, shareOf(res, "b").Rounded, shareOf(res, "c").Rounded)
	}
	// Exact shares kept for audit: 10,000,001/3 each.
	want := new(big.Rat).SetFrac64(10_000_001, 3)
	if shareOf(res, "a").Exact.Cmp(want) != 0 {
		t.Fatalf("exact share = %s, want %s", shareOf(res, "a").Exact.RatString(), want.RatString())
	}
}

// Fractional remainders (not ties) decide who receives the residue.
func TestLargestRemainderPrefersLargestFraction(t *testing.T) {
	// 1,000 over occupants 1,1,3 → exact 200,200,600; floors 200,200,600 Σ=1000.
	units := []Unit{{ID: "a", Occupants: 1}, {ID: "b", Occupants: 1}, {ID: "c", Occupants: 3}}
	res, err := Calculate(CostItem{ID: "x", Total: 1_000, Method: MethodPerOccupant}, units)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if sumRounded(res) != 1_000 {
		t.Fatalf("Σ = %d", sumRounded(res))
	}
	if shareOf(res, "c").Rounded != 600 || shareOf(res, "a").Rounded != 200 {
		t.Fatalf("shares wrong: a=%d c=%d", shareOf(res, "a").Rounded, shareOf(res, "c").Rounded)
	}

	// 1,001 over the same weights → floors 200,200,600 (Σ=1000), remainder 1
	// goes to the largest fractional remainder: a and b tie (0.0), c has 0.0
	// too — exact 1001/5=200.2, 1001/5=200.2, 3003/5=600.6 → c wins.
	res, err = Calculate(CostItem{ID: "x", Total: 1_001, Method: MethodPerOccupant}, units)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if sumRounded(res) != 1_001 {
		t.Fatalf("Σ = %d", sumRounded(res))
	}
	if shareOf(res, "c").Rounded != 601 {
		t.Fatalf("largest fraction should get residue: c=%d", shareOf(res, "c").Rounded)
	}
}

// --- per-area -------------------------------------------------------------------

func TestPerArea(t *testing.T) {
	units := []Unit{{ID: "a", AreaM2: 100}, {ID: "b", AreaM2: 300}}
	res, err := Calculate(CostItem{ID: "x", Total: 4_000, Method: MethodPerArea}, units)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if shareOf(res, "a").Rounded != 1_000 || shareOf(res, "b").Rounded != 3_000 {
		t.Fatalf("per-area shares wrong: a=%d b=%d", shareOf(res, "a").Rounded, shareOf(res, "b").Rounded)
	}
	if sumRounded(res) != 4_000 {
		t.Fatalf("Σ = %d", sumRounded(res))
	}
}

// --- fixed ----------------------------------------------------------------------

func TestFixed(t *testing.T) {
	units := []Unit{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	res, err := Calculate(CostItem{ID: "x", Method: MethodFixed, FixedAmountPerUnit: 500_000}, units)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	for _, id := range []string{"a", "b", "c"} {
		if shareOf(res, id).Rounded != 500_000 {
			t.Fatalf("%s = %d, want 500,000", id, shareOf(res, id).Rounded)
		}
	}
	if res.TotalUsed != 1_500_000 || !res.Reconciled {
		t.Fatalf("fixed total used = %d reconciled=%v", res.TotalUsed, res.Reconciled)
	}
}

// --- specific units -------------------------------------------------------------

func TestSpecificUnits(t *testing.T) {
	units := []Unit{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	res, err := Calculate(CostItem{ID: "x", Total: 1_000, Method: MethodSpecificUnits, UnitIDs: []string{"a", "c"}}, units)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if shareOf(res, "a").Rounded != 500 || shareOf(res, "c").Rounded != 500 {
		t.Fatalf("specific shares wrong: a=%d c=%d", shareOf(res, "a").Rounded, shareOf(res, "c").Rounded)
	}
	if shareOf(res, "b") != nil {
		t.Fatalf("non-selected unit must have no share row")
	}
	if sumRounded(res) != 1_000 {
		t.Fatalf("Σ = %d", sumRounded(res))
	}
}

// --- combined weights -------------------------------------------------------------

func TestCombined(t *testing.T) {
	// 10,000,000 = 50% equal + 30% per_area + 20% per_occupant over two units
	// (areas 100/300, occupants 1/3):
	//   equal pool 5,000,000    → 2,500,000 / 2,500,000
	//   area  pool 3,000,000    →   750,000 / 2,250,000
	//   occupant pool 2,000,000 →   500,000 / 1,500,000
	units := []Unit{{ID: "a", Occupants: 1, AreaM2: 100}, {ID: "b", Occupants: 3, AreaM2: 300}}
	res, err := Calculate(CostItem{
		ID:     "x",
		Total:  10_000_000,
		Method: MethodCombined,
		ComboWeights: []ComboWeight{
			{Method: MethodEqual, Weight: 50},
			{Method: MethodPerArea, Weight: 30},
			{Method: MethodPerOccupant, Weight: 20},
		},
	}, units)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if shareOf(res, "a").Rounded != 3_750_000 {
		t.Fatalf("a = %d, want 3,750,000", shareOf(res, "a").Rounded)
	}
	if shareOf(res, "b").Rounded != 6_250_000 {
		t.Fatalf("b = %d, want 6,250,000", shareOf(res, "b").Rounded)
	}
	if sumRounded(res) != 10_000_000 || !res.Reconciled {
		t.Fatalf("combined not reconciled: Σ=%d", sumRounded(res))
	}
}

// Combined pool allocation itself uses largest remainder when weights don't
// divide the total evenly (e.g. 33/33/34 of 100 Toman).
func TestCombinedWeightPoolReconciliation(t *testing.T) {
	units := []Unit{{ID: "a"}, {ID: "b"}}
	res, err := Calculate(CostItem{
		ID:     "x",
		Total:  100,
		Method: MethodCombined,
		ComboWeights: []ComboWeight{
			{Method: MethodEqual, Weight: 33},
			{Method: MethodFixed, Weight: 67, FixedAmountPerUnit: 33},
		},
	}, units)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	// equal pool 33 → 17/16 (tie-break); fixed pool 67 → 33+33=66…
	// Pools must sum to 100 and per-unit shares must sum to 100.
	if sumRounded(res) != 100 || !res.Reconciled {
		t.Fatalf("Σ = %d reconciled=%v", sumRounded(res), res.Reconciled)
	}
}

// --- zero participating factors (spec edge) ---------------------------------------

func TestZeroFactorErrors(t *testing.T) {
	cases := []struct {
		name  string
		item  CostItem
		units []Unit
		want  error
	}{
		{"equal no units", CostItem{Total: 100, Method: MethodEqual}, nil, ErrNoParticipants},
		{"equal all vacant", CostItem{Total: 100, Method: MethodEqual},
			[]Unit{{ID: "a", Vacant: true}}, ErrNoParticipants},
		{"per_occupant zero occupants", CostItem{Total: 100, Method: MethodPerOccupant},
			[]Unit{{ID: "a"}}, ErrZeroFactor},
		{"per_area zero area", CostItem{Total: 100, Method: MethodPerArea},
			[]Unit{{ID: "a"}}, ErrZeroFactor},
		{"fixed no units", CostItem{Method: MethodFixed, FixedAmountPerUnit: 100}, nil, ErrNoParticipants},
		{"specific empty", CostItem{Total: 100, Method: MethodSpecificUnits},
			[]Unit{{ID: "a"}}, ErrSpecificUnitsEmpty},
		{"combined no weights", CostItem{Total: 100, Method: MethodCombined},
			[]Unit{{ID: "a"}}, ErrBadWeights},
		{"combined weights != 100", CostItem{Total: 100, Method: MethodCombined,
			ComboWeights: []ComboWeight{{Method: MethodEqual, Weight: 50}}},
			[]Unit{{ID: "a"}}, ErrBadWeights},
		{"combined zero weight", CostItem{Total: 100, Method: MethodCombined,
			ComboWeights: []ComboWeight{{Method: MethodEqual, Weight: 0}, {Method: MethodFixed, Weight: 100}}},
			[]Unit{{ID: "a"}}, ErrBadWeights},
		{"combined nested", CostItem{Total: 100, Method: MethodCombined,
			ComboWeights: []ComboWeight{{Method: MethodCombined, Weight: 100}}},
			[]Unit{{ID: "a"}}, ErrBadWeights},
		{"non-positive total", CostItem{Total: 0, Method: MethodEqual},
			[]Unit{{ID: "a"}}, ErrNonPositiveTotal},
		{"non-positive fixed", CostItem{Method: MethodFixed, FixedAmountPerUnit: 0},
			[]Unit{{ID: "a"}}, ErrNonPositiveTotal},
		{"unknown method", CostItem{Total: 100, Method: "bogus"},
			[]Unit{{ID: "a"}}, ErrUnknownMethod},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Calculate(tc.item, tc.units)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// --- vacant-unit inclusion (BR-06/BR-07) --------------------------------------------

func TestVacantInclusion(t *testing.T) {
	units := []Unit{{ID: "a"}, {ID: "b", Vacant: true}}

	// Default: vacant excluded from the equal split.
	res, err := Calculate(CostItem{ID: "x", Total: 1_000, Method: MethodEqual}, units)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if shareOf(res, "b") != nil || shareOf(res, "a").Rounded != 1_000 {
		t.Fatalf("vacant unit must be excluded by default")
	}

	// include_vacant pulls it in.
	res, err = Calculate(CostItem{ID: "x", Total: 1_000, Method: MethodEqual, IncludeVacant: true}, units)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if shareOf(res, "a").Rounded != 500 || shareOf(res, "b").Rounded != 500 {
		t.Fatalf("include_vacant must include the vacant unit")
	}

	// A vacant unit with occupants (edge: counted but not resident) still
	// contributes to per_occupant only when included.
	units2 := []Unit{{ID: "a", Occupants: 2}, {ID: "b", Occupants: 2, Vacant: true}}
	res, err = Calculate(CostItem{ID: "x", Total: 800, Method: MethodPerOccupant}, units2)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if shareOf(res, "a").Rounded != 800 || shareOf(res, "b") != nil {
		t.Fatalf("vacant occupants must be excluded by default: a=%d", shareOf(res, "a").Rounded)
	}
	res, err = Calculate(CostItem{ID: "x", Total: 800, Method: MethodPerOccupant, IncludeVacant: true}, units2)
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if shareOf(res, "a").Rounded != 400 || shareOf(res, "b").Rounded != 400 {
		t.Fatalf("include_vacant per_occupant wrong: a=%d b=%d",
			shareOf(res, "a").Rounded, shareOf(res, "b").Rounded)
	}
}

// --- late fees (FR-016) --------------------------------------------------------------

func TestLateFee(t *testing.T) {
	cases := []struct {
		name     string
		kind     LateFeeType
		value    float64
		base     int64
		daysLate int
		want     int64
		wantErr  bool
	}{
		{"none", LateFeeNone, 0, 2_600_000, 10, 0, false},
		{"fixed", LateFeeFixed, 50_000, 2_600_000, 0, 50_000, false},
		{"percent", LateFeePercent, 2.5, 2_600_000, 0, 65_000, false},
		{"percent floors", LateFeePercent, 2.5, 2_600_001, 0, 65_000, false}, // 65000.025 → 65000
		{"per_day", LateFeePerDay, 10_000, 2_600_000, 3, 30_000, false},
		{"per_day zero days", LateFeePerDay, 10_000, 2_600_000, 0, 0, false},
		{"negative days clamped", LateFeePerDay, 10_000, 2_600_000, -5, 0, false},
		{"unknown kind", LateFeeType("bogus"), 1, 100, 0, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LateFee(tc.kind, tc.value, tc.base, tc.daysLate)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("late fee: %v", err)
			}
			if got != tc.want {
				t.Fatalf("late fee = %d, want %d", got, tc.want)
			}
		})
	}
}
