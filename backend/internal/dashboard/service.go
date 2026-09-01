package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/announcement"
	"hamsa/internal/auth"
	"hamsa/internal/billing"
	"hamsa/internal/maintenance"
	"hamsa/internal/platform/httpx"
)

// Persian user-facing messages (contracts/api.md: server messages are Persian).
const (
	msgUnauthorized = "برای این عملیات باید وارد شوید"
	msgForbidden    = "دسترسی غیرمجاز است"
)

// Clock abstracts time so tests can pin the "current month" anchor.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// Service owns the read-only aggregation queries backing US9 (`/me/home`) and
// US10 (`/buildings/{id}/dashboard`). No mutations happen here.
type Service struct {
	db         *gorm.DB
	scopes     *auth.ScopeResolver
	annRepo    *announcement.Repository
	maintRepo  *maintenance.Repository
	clock      Clock
}

// NewService wires the dashboard aggregator. It reads through the
// announcement module's repository (audience resolution + UnreadCount);
// the remaining dependencies are required.
func NewService(
	db *gorm.DB,
	scopes *auth.ScopeResolver,
	annRepo *announcement.Repository,
	maintRepo *maintenance.Repository,
) *Service {
	return &Service{
		db:        db,
		scopes:    scopes,
		annRepo:   annRepo,
		maintRepo: maintRepo,
		clock:     realClock{},
	}
}

// SetClock overrides the clock (used in tests).
func (s *Service) SetClock(c Clock) { s.clock = c }

// --- US9: resident home -----------------------------------------------------

// Home returns the aggregated payload for GET /me/home (contracts/api.md
// "Resident Panel P0-08"). The response is strictly scoped to the calling
// resident's active units.
func (s *Service) Home(ctx context.Context, resident *auth.User) (*HomeSummary, error) {
	if resident == nil {
		return nil, httpx.Unauthorized(msgUnauthorized)
	}

	unitIDs, err := s.scopes.ResidentUnitIDs(ctx, resident.Phone)
	if err != nil {
		return nil, err
	}

	out := &HomeSummary{
		UnitCount:           len(unitIDs),
		GeneratedAt:         s.clock.Now(),
		LatestAnnouncements: []LatestAnnouncement{},
	}

	if len(unitIDs) == 0 {
		// Resident without active units still gets a well-formed response.
		return out, nil
	}

	// 1. Payable amount + latest invoice across the resident's units.
	payable, latest, err := s.payableAndLatestInvoice(ctx, unitIDs)
	if err != nil {
		return nil, err
	}
	out.PayableAmount = billing.Money(payable)
	out.LatestInvoice = latest

	// 2. Open request count + latest request status (resident's own requests).
	openCount, latestReq, err := s.openRequestSummary(ctx, resident.ID)
	if err != nil {
		return nil, err
	}
	out.OpenRequestCount = openCount
	out.LatestRequest = latestReq

	// 3. Latest announcements + unread count (US8 audience logic reused).
	latestAnn, unread, err := s.announcementSummary(ctx, resident)
	if err != nil {
		return nil, err
	}
	out.LatestAnnouncements = latestAnn
	out.UnreadAnnouncementCount = unread

	return out, nil
}

func (s *Service) payableAndLatestInvoice(
	ctx context.Context,
	unitIDs []uuid.UUID,
) (int64, *LatestInvoice, error) {
	// unpaid + partial + expired invoices (expired stays in — spec §4).
	type invRow struct {
		ID          uuid.UUID
		InvoiceNo   string
		UnitID      uuid.UUID
		UnitNumber  *string
		PeriodID    uuid.UUID
		PeriodTitle *string
		FinalAmount int64
		PaidAmount  int64
		DueDate     *time.Time
		Status      string
		IssueDate   *time.Time
		CreatedAt   time.Time
	}
	var rows []invRow
	if err := s.db.WithContext(ctx).
		Table("invoices i").
		Select(`i.id, i.invoice_number, i.unit_id, un.number AS unit_number,
		        i.period_id, bp.title AS period_title,
		        i.final_amount, i.paid_amount, i.due_date, i.status, i.issue_date,
		        i.created_at`).
		Joins("JOIN units un ON un.id = i.unit_id").
		Joins("JOIN billing_periods bp ON bp.id = i.period_id").
		Where("i.unit_id IN ?", unitIDs).
		Where("i.status IN ?", []string{billing.InvoiceUnpaid, billing.InvoicePartial, billing.InvoiceExpired}).
		Order("i.issue_date DESC NULLS LAST, i.created_at DESC").
		Find(&rows).Error; err != nil {
		return 0, nil, fmt.Errorf("query resident invoices: %w", err)
	}

	var payable int64
	var newest *LatestInvoice
	for i := range rows {
		outstanding := rows[i].FinalAmount - rows[i].PaidAmount
		if outstanding < 0 {
			outstanding = 0
		}
		payable += outstanding

		if newest == nil {
			li := &LatestInvoice{
				ID:          rows[i].ID.String(),
				InvoiceNo:   rows[i].InvoiceNo,
				FinalAmount: billing.Money(rows[i].FinalAmount),
				Status:      rows[i].Status,
			}
			if rows[i].UnitNumber != nil {
				li.UnitNumber = *rows[i].UnitNumber
			}
			if rows[i].PeriodTitle != nil {
				li.PeriodTitle = *rows[i].PeriodTitle
			}
			if rows[i].DueDate != nil {
				li.DueDate = rows[i].DueDate.Format("2006-01-02")
			}
			newest = li
		}
	}
	return payable, newest, nil
}

func (s *Service) openRequestSummary(
	ctx context.Context,
	residentID uuid.UUID,
) (int, *LatestRequest, error) {
	if s.maintRepo == nil {
		return 0, nil, nil
	}
	openCount, err := s.maintRepo.CountOpenForUser(ctx, residentID)
	if err != nil {
		return 0, nil, fmt.Errorf("count open requests: %w", err)
	}

	latest, err := s.maintRepo.LatestForUser(ctx, residentID)
	if err != nil {
		return 0, nil, fmt.Errorf("latest request: %w", err)
	}
	if latest == nil {
		return openCount, nil, nil
	}
	return openCount, &LatestRequest{
		ID:        latest.ID.String(),
		Title:     latest.Title,
		Priority:  latest.Priority,
		Status:    latest.Status,
		CreatedAt: latest.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Service) announcementSummary(
	ctx context.Context,
	resident *auth.User,
) ([]LatestAnnouncement, int, error) {
	if s.annRepo == nil {
		return []LatestAnnouncement{}, 0, nil
	}
	residentUnits, err := s.annRepo.ResidentUnits(ctx, resident.Phone)
	if err != nil {
		return nil, 0, fmt.Errorf("resolve announcement units: %w", err)
	}
	now := s.clock.Now()

	// Unread count uses the existing repo helper (unaffected by page size).
	var unread int64
	unread, err = s.annRepo.UnreadCountForResident(ctx, residentUnits, resident.ID, now)
	if err != nil {
		return nil, 0, fmt.Errorf("unread count: %w", err)
	}

	// Most recent 5 visible announcements.
	visible, err := s.annRepo.VisibleAnnouncements(ctx, residentUnits, now)
	if err != nil {
		return nil, 0, fmt.Errorf("list visible announcements: %w", err)
	}

	// Hydrate is_read from the resident's read-tracking rows.
	reads, err := s.annRepo.ReadsForUser(ctx, resident.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("list announcement reads: %w", err)
	}
	for i := range visible {
		if reads[visible[i].ID] {
			visible[i].IsRead = true
		}
	}
	if len(visible) > 1 {
		sortAnnouncementsNewestFirst(visible)
	}
	limit := 5
	if len(visible) < limit {
		limit = len(visible)
	}
	out := make([]LatestAnnouncement, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, LatestAnnouncement{
			ID:        visible[i].ID.String(),
			Title:     visible[i].Title,
			CreatedAt: visible[i].CreatedAt.Format(time.RFC3339),
			IsRead:    visible[i].IsRead,
		})
	}
	return out, int(unread), nil
}

// --- US10: manager building dashboard --------------------------------------

// BuildingDashboard returns the cards + alerts + quick-actions payload for
// GET /buildings/{id}/dashboard.
func (s *Service) BuildingDashboard(
	ctx context.Context,
	manager *auth.User,
	buildingID uuid.UUID,
) (*BuildingDashboard, error) {
	if manager == nil {
		return nil, httpx.Unauthorized(msgUnauthorized)
	}
	ok, err := s.scopes.CanManagerAccessBuilding(ctx, manager.ID, buildingID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgForbidden)
	}

	now := s.clock.Now()
	monthStart := monthFloor(now)
	monthEnd := monthStart.AddDate(0, 1, 0)

	out := &BuildingDashboard{

		BuildingID:   buildingID.String(),
		Month:        monthStart.Format("2006-01"),
		GeneratedAt:  now,
		Alerts:       []DashboardAlert{},
		QuickActions: defaultQuickActions(buildingID),
	}

	if err := s.db.WithContext(ctx).
		Table("units").
		Where("building_id = ? AND deleted_at IS NULL", buildingID).
		Count(&out.UnitCount).Error; err != nil {
		return nil, fmt.Errorf("unit count: %w", err)
	}
	if err := s.db.WithContext(ctx).
		Table("units").
		Where("building_id = ? AND deleted_at IS NULL AND status = ?", buildingID, "occupied").
		Count(&out.OccupiedUnitCount).Error; err != nil {
		return nil, fmt.Errorf("occupied unit count: %w", err)
	}

	// Total debt + debtor count (unit_balances).
	var totalDebt int64
	if err := s.db.WithContext(ctx).
		Table("unit_balances ub").
		Select("COALESCE(SUM(GREATEST(ub.balance, 0)), 0)").
		Joins("JOIN units un ON un.id = ub.unit_id").
		Where("un.building_id = ? AND un.deleted_at IS NULL", buildingID).
		Scan(&totalDebt).Error; err != nil {
		return nil, fmt.Errorf("total debt: %w", err)
	}
	out.TotalDebt = billing.Money(totalDebt)
	if err := s.db.WithContext(ctx).
		Table("unit_balances ub").
		Joins("JOIN units un ON un.id = ub.unit_id").
		Where("un.building_id = ? AND un.deleted_at IS NULL", buildingID).
		Where("ub.balance > 0").
		Count(&out.DebtorUnitCount).Error; err != nil {
		return nil, fmt.Errorf("debtor count: %w", err)
	}

	// Month income (verified payments) and month expense.
	var monthIncome int64
	if err := s.db.WithContext(ctx).
		Table("payments").
		Where("building_id = ?", buildingID).
		Where("status = ?", "verified").
		Where("paid_at >= ? AND paid_at < ?", monthStart, monthEnd).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&monthIncome).Error; err != nil {
		return nil, fmt.Errorf("month income: %w", err)
	}
	out.MonthIncome = billing.Money(monthIncome)
	var monthExpense int64
	if err := s.db.WithContext(ctx).
		Table("expenses").
		Where("building_id = ? AND deleted_at IS NULL", buildingID).
		Where("expense_date >= ? AND expense_date < ?", monthStart, monthEnd).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&monthExpense).Error; err != nil {
		return nil, fmt.Errorf("month expense: %w", err)
	}
	out.MonthExpense = billing.Money(monthExpense)

	// Open maintenance requests.
	if err := s.db.WithContext(ctx).
		Table("maintenance_requests").
		Where("building_id = ?", buildingID).
		Where("status IN ?", []string{
			maintenance.StatusNew,
			maintenance.StatusUnderReview,
			maintenance.StatusInProgress,
			maintenance.StatusDone,
		}).
		Count(&out.OpenRequests).Error; err != nil {
		return nil, fmt.Errorf("open requests: %w", err)
	}

	// Pending expenses (approval_status = pending).
	if err := s.db.WithContext(ctx).
		Table("expenses").
		Where("building_id = ? AND deleted_at IS NULL", buildingID).
		Where("approval_status = ?", "pending").
		Count(&out.PendingExpenses).Error; err != nil {
		return nil, fmt.Errorf("pending expenses: %w", err)
	}

	alerts, err := s.alerts(ctx, buildingID, now)
	if err != nil {
		return nil, err
	}
	out.Alerts = alerts

	return out, nil
}

func (s *Service) alerts(
	ctx context.Context,
	buildingID uuid.UUID,
	now time.Time,
) ([]DashboardAlert, error) {
	const perKindLimit = 10
	out := []DashboardAlert{}

	type debtorRow struct {
		UnitID     uuid.UUID
		UnitNumber string
		Balance    int64
	}
	var debtors []debtorRow
	if err := s.db.WithContext(ctx).
		Table("unit_balances ub").
		Select("ub.unit_id, un.number AS unit_number, ub.balance").
		Joins("JOIN units un ON un.id = ub.unit_id").
		Where("un.building_id = ? AND un.deleted_at IS NULL", buildingID).
		Where("ub.balance > 0").
		Order("ub.balance DESC").
		Limit(perKindLimit).
		Scan(&debtors).Error; err != nil {
		return nil, fmt.Errorf("alerts.debtors: %w", err)
	}
	for _, d := range debtors {
		out = append(out, DashboardAlert{
			Kind:       "debtor_unit",
			Severity:   1,
			UnitID:     d.UnitID.String(),
			UnitNumber: d.UnitNumber,
			Amount:     billing.Money(d.Balance),
			Title:      "واحد بدهکار",
			RefType:    "unit",
			RefID:      d.UnitID.String(),
		})
	}

	type pastDueRow struct {
		ID          uuid.UUID
		UnitID      uuid.UUID
		UnitNumber  string
		FinalAmount int64
		PaidAmount  int64
		DueDate     time.Time
	}
	var pastDue []pastDueRow
	if err := s.db.WithContext(ctx).
		Table("invoices i").
		Select("i.id, i.unit_id, un.number AS unit_number, i.final_amount, i.paid_amount, i.due_date").
		Joins("JOIN units un ON un.id = i.unit_id").
		Where("un.building_id = ? AND un.deleted_at IS NULL", buildingID).
		Where("i.due_date IS NOT NULL AND i.due_date < ?", now).
		Where("i.status IN ?", []string{billing.InvoiceUnpaid, billing.InvoicePartial}).
		Order("i.due_date ASC").
		Limit(perKindLimit).
		Scan(&pastDue).Error; err != nil {
		return nil, fmt.Errorf("alerts.past_due: %w", err)
	}
	for _, r := range pastDue {
		outstanding := r.FinalAmount - r.PaidAmount
		if outstanding < 0 {
			outstanding = 0
		}
		out = append(out, DashboardAlert{
			Kind:       "past_due_invoice",
			Severity:   2,
			UnitID:     r.UnitID.String(),
			UnitNumber: r.UnitNumber,
			Amount:     billing.Money(outstanding),
			Title:      "صورتحساب سررسید گذشته",
			RefType:    "invoice",
			RefID:      r.ID.String(),
			CreatedAt:  r.DueDate.Format("2006-01-02"),
		})
	}

	type reqRow struct {
		ID        uuid.UUID
		Title     string
		Status    string
		Priority  string
		CreatedAt time.Time
	}
	var reqs []reqRow
	if err := s.db.WithContext(ctx).
		Table("maintenance_requests").
		Select("id, title, status, priority, created_at").
		Where("building_id = ?", buildingID).
		Where("status IN ?", []string{
			maintenance.StatusNew,
			maintenance.StatusUnderReview,
			maintenance.StatusInProgress,
			maintenance.StatusDone,
		}).
		Order("created_at DESC").
		Limit(perKindLimit).
		Scan(&reqs).Error; err != nil {
		return nil, fmt.Errorf("alerts.open_requests: %w", err)
	}
	for _, r := range reqs {
		sev := 0
		if r.Priority == maintenance.PriUrgent {
			sev = 2
		} else if r.Priority == maintenance.PriImportant {
			sev = 1
		}
		out = append(out, DashboardAlert{
			Kind:      "open_request",
			Severity:  sev,
			Title:     r.Title,
			RefType:   "maintenance_request",
			RefID:     r.ID.String(),
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		})
	}

	type expRow struct {
		ID          uuid.UUID
		Title       string
		Amount      int64
		ExpenseDate time.Time
	}
	var exps []expRow
	if err := s.db.WithContext(ctx).
		Table("expenses").
		Select("id, title, amount, expense_date").
		Where("building_id = ? AND deleted_at IS NULL", buildingID).
		Where("approval_status = ?", "pending").
		Order("expense_date DESC").
		Limit(perKindLimit).
		Scan(&exps).Error; err != nil {
		return nil, fmt.Errorf("alerts.pending_expenses: %w", err)
	}
	for _, r := range exps {
		out = append(out, DashboardAlert{
			Kind:      "pending_expense",
			Severity:  1,
			Title:     r.Title,
			Amount:    billing.Money(r.Amount),
			RefType:   "expense",
			RefID:     r.ID.String(),
			CreatedAt: r.ExpenseDate.Format("2006-01-02"),
		})
	}

	if len(out) > 20 {
		out = out[:20]
	}
	return out, nil
}

func defaultQuickActions(buildingID uuid.UUID) []ManagerQuickAction {
	bid := buildingID.String()
	return []ManagerQuickAction{
		{Key: "issue_charge", Title: "صدور شارژ", Path: "/manager/periods/" + bid + "/new"},
		{Key: "record_expense", Title: "ثبت هزینه", Path: "/manager/expenses/" + bid + "/new"},
		{Key: "record_payment", Title: "ثبت پرداخت", Path: "/manager/ledger/" + bid},
		{Key: "send_announcement", Title: "انتشار اطلاعیه", Path: "/manager/announcements/" + bid + "/new"},
		{Key: "open_requests", Title: "درخواست‌ها", Path: "/manager/maintenance/" + bid},
	}
}

func monthFloor(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func sortAnnouncementsNewestFirst(arr []announcement.AnnouncementWithRead) {
	for i := 1; i < len(arr); i++ {
		for j := i; j > 0 && arr[j-1].CreatedAt.Before(arr[j].CreatedAt); j-- {
			arr[j-1], arr[j] = arr[j], arr[j-1]
		}
	}
}
