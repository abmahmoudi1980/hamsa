// Package dashboard serves US9 resident self-service (`GET /me/home`) and
// US10 manager dashboard (`GET /buildings/{id}/dashboard`). Both endpoints
// are pure aggregation reads — no mutations live here. The package depends
// on the platform gorm handle plus small per-domain projections; we deliberately
// keep it thin so the resident panel and the manager overview can render in a
// single round trip without loading every underlying record.
package dashboard

import "time"

// LatestInvoice is the resident's most recent invoice row (US9 T082 / FR-034).
// `FinalAmount` and `DueDate` are rendered Jalali by the client; the wire
// stays ISO-8601 + integer Toman string per contracts/api.md.
type LatestInvoice struct {
	ID           string `json:"id"`
	InvoiceNo    string `json:"invoice_number"`
	UnitNumber   string `json:"unit_number,omitempty"`
	PeriodTitle  string `json:"period_title,omitempty"`
	FinalAmount  int64  `json:"final_amount"`
	DueDate      string `json:"due_date,omitempty"`
	Status       string `json:"status"`
}

// LatestRequest is the resident's newest maintenance request summary.
type LatestRequest struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Priority   string `json:"priority"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// LatestAnnouncement is the most recent announcement targeted at the resident.
type LatestAnnouncement struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at,omitempty"`
	IsRead    bool   `json:"is_read"`
}

// HomeSummary is the response payload of GET /me/home (contracts/api.md
// "Resident Panel P0-08"). `PayableAmount` is the sum of unpaid + partial
// invoice final-amounts across the resident's active units; `UnitCount` is the
// number of units the resident currently occupies (0 when none — empty
// sections then read as "no data" rather than as errors).
type HomeSummary struct {
	PayableAmount     int64                `json:"payable_amount"`
	UnitCount         int                  `json:"unit_count"`
	LatestInvoice     *LatestInvoice       `json:"latest_invoice,omitempty"`
	OpenRequestCount  int                  `json:"open_request_count"`
	LatestRequest     *LatestRequest       `json:"latest_request,omitempty"`
	LatestAnnouncements []LatestAnnouncement `json:"latest_announcements"`
	UnreadAnnouncementCount int            `json:"unread_announcement_count"`
	GeneratedAt       time.Time            `json:"generated_at"`
}

// ManagerQuickAction is the deep-link target list returned by the manager
// dashboard (FR-035).
type ManagerQuickAction struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

// DashboardAlert is a single alert row (debtor / past-due / open request /
// pending expense). Severity 0 = info, 1 = warn, 2 = danger.
type DashboardAlert struct {
	Kind        string `json:"kind"`        // debtor_unit | past_due_invoice | open_request | pending_expense
	Severity    int    `json:"severity"`    // 0=info, 1=warn, 2=danger
	UnitID      string `json:"unit_id,omitempty"`
	UnitNumber  string `json:"unit_number,omitempty"`
	Amount      int64  `json:"amount,omitempty"`
	Title       string `json:"title,omitempty"`
	RefType     string `json:"ref_type,omitempty"`
	RefID       string `json:"ref_id,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// BuildingDashboard is the response payload of GET /buildings/{id}/dashboard.
type BuildingDashboard struct {
	BuildingID      string               `json:"building_id"`
	UnitCount       int64                `json:"unit_count"`
	OccupiedUnitCount int64              `json:"occupied_unit_count"`
	DebtorUnitCount   int64              `json:"debtor_unit_count"`
	TotalDebt       int64                `json:"total_debt"`
	MonthIncome     int64                `json:"month_income"`
	MonthExpense    int64                `json:"month_expense"`
	OpenRequests    int64                `json:"open_requests"`
	PendingExpenses int64                `json:"pending_expenses"`
	Month           string               `json:"month"` // YYYY-MM (Gregorian anchor — client renders Jalali)
	Alerts          []DashboardAlert     `json:"alerts"`
	QuickActions    []ManagerQuickAction `json:"quick_actions"`
	GeneratedAt     time.Time            `json:"generated_at"`
}
