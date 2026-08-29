package payment

// T056 — GORM repository for payments, unit balances, and the invoice reads
// the payment flow needs (row-locked invoices with adjustment sums).

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/building"
)

// Repository provides payment persistence.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a payment repository backed by db.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// --- invoices (read + lock) ---------------------------------------------------------

// ledgerInvoice is the payment-relevant projection of an invoice plus its
// adjustment sums. effective() is what the unit actually owes: the frozen
// final amount corrected by the explicit debit/credit notes (FR-017).
type ledgerInvoice struct {
	ID            uuid.UUID `gorm:"column:id"`
	BuildingID    uuid.UUID `gorm:"column:building_id"`
	UnitID        uuid.UUID `gorm:"column:unit_id"`
	InvoiceNumber string    `gorm:"column:invoice_number"`
	Status        string    `gorm:"column:status"`
	PaidAmount    Money     `gorm:"column:paid_amount"`
	FinalAmount   Money     `gorm:"column:final_amount"`
	DebitAdj      Money     `gorm:"column:debit_adj"`
	CreditAdj     Money     `gorm:"column:credit_adj"`
}

// effective returns the corrected payable amount of the invoice.
func (l *ledgerInvoice) effective() Money {
	return l.FinalAmount + l.DebitAdj - l.CreditAdj
}

// InvoiceForUpdate loads an invoice with a row lock (FOR UPDATE when inside a
// transaction) plus its adjustment sums; gorm.ErrRecordNotFound when missing.
// The lock is taken on the invoice row directly — PostgreSQL forbids FOR
// UPDATE on aggregate queries, so the adjustment sums are a second read
// inside the same transaction/lock window.
func (r *Repository) InvoiceForUpdate(ctx context.Context, tx *gorm.DB, invoiceID uuid.UUID) (*ledgerInvoice, error) {
	dbh := r.db
	if tx != nil {
		dbh = tx
	}
	var locked struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := dbh.WithContext(ctx).Raw(
		`SELECT id FROM invoices WHERE id = ? FOR UPDATE`, invoiceID,
	).Scan(&locked).Error; err != nil {
		return nil, err
	}
	if locked.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	var li ledgerInvoice
	err := dbh.WithContext(ctx).Raw(`
		SELECT i.id, i.building_id, i.unit_id, i.invoice_number, i.status, i.paid_amount, i.final_amount,
		       COALESCE(SUM(CASE WHEN a.kind = 'debit'  THEN a.amount ELSE 0 END), 0) AS debit_adj,
		       COALESCE(SUM(CASE WHEN a.kind = 'credit' THEN a.amount ELSE 0 END), 0) AS credit_adj
		FROM invoices i
		LEFT JOIN invoice_adjustments a ON a.invoice_id = i.id
		WHERE i.id = ?
		GROUP BY i.id
	`, invoiceID).Scan(&li).Error
	if err != nil {
		return nil, err
	}
	if li.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &li, nil
}

// ApplyInvoicePaid advances paid_amount and sets the derived status on an
// invoice (payment-driven mutation — the only fields allowed to change after
// issuance per contracts/api.md guarantee 1).
func (r *Repository) ApplyInvoicePaid(ctx context.Context, tx *gorm.DB, invoiceID uuid.UUID, newPaid Money, status string) error {
	dbh := r.db
	if tx != nil {
		dbh = tx
	}
	return dbh.WithContext(ctx).Exec(
		`UPDATE invoices SET paid_amount = ?, status = ?, updated_at = now() WHERE id = ?`,
		newPaid, status, invoiceID,
	).Error
}

// --- payments -----------------------------------------------------------------------

// CreatePayment inserts a payment row.
func (r *Repository) CreatePayment(ctx context.Context, p *Payment) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// GetPayment returns one payment or gorm.ErrRecordNotFound.
func (r *Repository) GetPayment(ctx context.Context, id uuid.UUID) (*Payment, error) {
	var p Payment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPaymentByAuthority resolves the in-flight gateway payment by its
// provider authority (callback lookup — the callback content is never
// trusted for identification or amount).
func (r *Repository) GetPaymentByAuthority(ctx context.Context, authority string) (*Payment, error) {
	var p Payment
	err := r.db.WithContext(ctx).
		Where("authority = ? AND status = ?", authority, StatusRecorded).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SavePayment persists status changes.
func (r *Repository) SavePayment(ctx context.Context, p *Payment) error {
	return r.db.WithContext(ctx).Save(p).Error
}

// PaymentFilter narrows payment lists (manager ledger + /me/payments).
type PaymentFilter struct {
	BuildingID uuid.UUID   // manager ledger scope
	UnitID     uuid.UUID   // single unit (ledger ?unit_id=)
	UnitIDs    []uuid.UUID // resident scope (/me/payments)
	From       *time.Time  // paid_at >= From
	To         *time.Time  // paid_at <= To
	Method     string
	Page       int
	Size       int
}

// ListPayments returns one filtered page of payments, newest first, with the
// unit number for display.
func (r *Repository) ListPayments(ctx context.Context, f PaymentFilter) ([]Payment, int64, error) {
	q := r.db.WithContext(ctx).Model(&Payment{})
	switch {
	case len(f.UnitIDs) > 0:
		q = q.Where("unit_id IN ?", f.UnitIDs)
	case f.UnitID != uuid.Nil:
		q = q.Where("unit_id = ?", f.UnitID)
	case f.BuildingID != uuid.Nil:
		q = q.Where("building_id = ?", f.BuildingID)
	}
	if f.Method != "" {
		q = q.Where("method = ?", f.Method)
	}
	if f.From != nil {
		q = q.Where("paid_at >= ?", f.From.Format("2006-01-02"))
	}
	if f.To != nil {
		q = q.Where("paid_at <= ?", f.To.Format("2006-01-02"))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := f.Page, f.Size
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var out []Payment
	err := q.
		Select("payments.*, (SELECT number FROM units u WHERE u.id = payments.unit_id) AS unit_number").
		Order("payments.created_at DESC").
		Limit(size).Offset((page - 1) * size).
		Find(&out).Error
	return out, total, err
}

// OccupantUserIDs returns the user ids of the unit's active occupants
// (notification recipients, FR-032).
func (r *Repository) OccupantUserIDs(ctx context.Context, unitID uuid.UUID) ([]uuid.UUID, error) {
	var out []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT u.id
		FROM users u
		JOIN persons p ON p.phone = u.phone AND p.deleted_at IS NULL
		JOIN occupancies o ON o.person_id = p.id
		WHERE o.unit_id = ? AND o.end_date IS NULL AND o.is_active
	`, unitID).Scan(&out).Error
	return out, err
}

// --- scope helpers --------------------------------------------------------------------

// ResidentUnitIDs returns the units the user's phone currently occupies
// (persons.phone -> active occupancies) — the /me scope and the
// object-level residency check (FR-037).
func (r *Repository) ResidentUnitIDs(ctx context.Context, phone string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT o.unit_id
		FROM occupancies o
		JOIN persons p ON p.id = o.person_id
		WHERE p.phone = ? AND p.deleted_at IS NULL AND o.end_date IS NULL`,
		phone).Scan(&ids).Error
	return ids, err
}

// isManagerOf reports whether the user holds a user_buildings grant.
func (r *Repository) isManagerOf(ctx context.Context, userID, buildingID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&building.UserBuilding{}).
		Where("user_id = ? AND building_id = ?", userID, buildingID).
		Count(&n).Error
	return n > 0, err
}

// UnitByID returns a unit (scope checks handled by callers).
func (r *Repository) UnitByID(ctx context.Context, unitID uuid.UUID) (*building.Unit, error) {
	var u building.Unit
	if err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", unitID).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// --- balances -------------------------------------------------------------------------

type balanceAggregates struct {
	Prior     int64 `gorm:"column:prior"`
	Current   int64 `gorm:"column:current_amt"`
	Late      int64 `gorm:"column:late"`
	Paid      int64 `gorm:"column:paid"`
	Credit    int64 `gorm:"column:credit"`
	DebitAdj  int64 `gorm:"column:debit_adj"`
	CreditAdj int64 `gorm:"column:credit_adj"`
	Verified  int64 `gorm:"column:verified_paid"`

	// Absorbed basis: paid_amount and embedded credit across ALL
	// non-cancelled invoices — fully paid ones included (their cash was
	// absorbed even though they left the open set).
	AbsorbedPaid   int64 `gorm:"column:absorbed_paid"`
	AbsorbedCredit int64 `gorm:"column:absorbed_credit"`
}

// balanceAggregatesQuery derives the §9 components from source tables.
//
//   - open invoices (unpaid/partial/expired) carry the unit's live debt;
//     cancelled invoices and their adjustments are excluded entirely.
//   - Σ(final − prior) over open invoices handles carried balances: a newer
//     invoice's prior_debt embeds the older outstanding, so it must not be
//     double-counted.
//   - credit_asset = verified payments − what invoices absorbed (paid_amount
//   - embedded credit_amount) — surplus overpayment held for the unit,
//     snapshot into the next invoice's credit_amount at calculation.
const balanceAggregatesQuery = `
WITH nc AS (
    SELECT * FROM invoices WHERE unit_id = ? AND status <> 'cancelled'
), open_inv AS (
    SELECT * FROM nc WHERE status IN ('unpaid', 'partial', 'expired')
)
SELECT
    (SELECT COALESCE(SUM(prior_debt), 0)      FROM open_inv) AS prior,
    (SELECT COALESCE(SUM(base_amount), 0)     FROM open_inv) AS current_amt,
    (SELECT COALESCE(SUM(late_fee_amount), 0) FROM open_inv) AS late,
    (SELECT COALESCE(SUM(paid_amount), 0)     FROM open_inv) AS paid,
    (SELECT COALESCE(SUM(credit_amount), 0)   FROM open_inv) AS credit,
    (SELECT COALESCE(SUM(CASE WHEN a.kind = 'debit'  THEN a.amount ELSE 0 END), 0)
       FROM invoice_adjustments a JOIN nc i ON i.id = a.invoice_id)                    AS debit_adj,
    (SELECT COALESCE(SUM(CASE WHEN a.kind = 'credit' THEN a.amount ELSE 0 END), 0)
       FROM invoice_adjustments a JOIN nc i ON i.id = a.invoice_id)                    AS credit_adj,
    (SELECT COALESCE(SUM(amount), 0) FROM payments
       WHERE unit_id = ? AND status = 'verified')                                      AS verified_paid,
    (SELECT COALESCE(SUM(paid_amount), 0)   FROM nc) AS absorbed_paid,
    (SELECT COALESCE(SUM(credit_amount), 0) FROM nc) AS absorbed_credit
`

// recomputeBalance derives the unit's position and upserts the unit_balances
// row (idempotent — recomputation from source data cannot drift). tx may be
// nil; when non-nil the recompute joins the caller's transaction.
func (r *Repository) recomputeBalance(ctx context.Context, tx *gorm.DB, unitID uuid.UUID) (*UnitBalance, error) {
	dbh := r.db
	if tx != nil {
		dbh = tx
	}
	var agg balanceAggregates
	if err := dbh.WithContext(ctx).Raw(balanceAggregatesQuery, unitID, unitID).Scan(&agg).Error; err != nil {
		return nil, err
	}

	ub := &UnitBalance{
		UnitID:               unitID,
		PriorDebt:            agg.Prior + agg.DebitAdj,
		CurrentInvoiceAmount: agg.Current,
		LateFeeTotal:         agg.Late,
		PaidTotal:            agg.Paid,
		Credit:               agg.Credit + agg.CreditAdj,
	}
	// §9: balance = prior_debt + current + late_fee − paid − credit. A
	// negative result would mean unrecorded surplus; clamp defensively (the
	// surplus belongs in credit_asset, which is derived below).
	balance := ub.PriorDebt + ub.CurrentInvoiceAmount + ub.LateFeeTotal - ub.PaidTotal - ub.Credit
	if balance < 0 {
		balance = 0
	}
	ub.Balance = balance

	// Surplus money: verified payments beyond what invoices absorbed (paid
	// amounts and embedded credit across ALL non-cancelled invoices — fully
	// paid ones included). The embedded credit_amount is subtracted because
	// it is already inside final_amount (consumed credit).
	absorbed := agg.AbsorbedPaid + agg.AbsorbedCredit
	asset := agg.Verified - absorbed
	if asset < 0 {
		asset = 0
	}
	ub.CreditAsset = asset
	ub.UpdatedAt = time.Now()

	err := dbh.WithContext(ctx).Exec(`
		INSERT INTO unit_balances (unit_id, prior_debt, current_invoice_amount, late_fee_total,
		                           credit, paid_total, credit_asset, balance, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, now())
		ON CONFLICT (unit_id) DO UPDATE SET
		    prior_debt = EXCLUDED.prior_debt,
		    current_invoice_amount = EXCLUDED.current_invoice_amount,
		    late_fee_total = EXCLUDED.late_fee_total,
		    credit = EXCLUDED.credit,
		    paid_total = EXCLUDED.paid_total,
		    credit_asset = EXCLUDED.credit_asset,
		    balance = EXCLUDED.balance,
		    updated_at = now()
	`, ub.UnitID, ub.PriorDebt, ub.CurrentInvoiceAmount, ub.LateFeeTotal,
		ub.Credit, ub.PaidTotal, ub.CreditAsset, ub.Balance).Error
	if err != nil {
		return nil, err
	}
	return ub, nil
}

// GetUnitBalance returns the stored balance row or gorm.ErrRecordNotFound
// when the unit has no financial history yet.
func (r *Repository) GetUnitBalance(ctx context.Context, unitID uuid.UUID) (*UnitBalance, error) {
	var ub UnitBalance
	err := r.db.WithContext(ctx).Where("unit_id = ?", unitID).First(&ub).Error
	if err != nil {
		return nil, err
	}
	return &ub, nil
}
