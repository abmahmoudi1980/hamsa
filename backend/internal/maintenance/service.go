package maintenance

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/audit"
	"hamsa/internal/auth"
	"hamsa/internal/notification"
	"hamsa/internal/platform/httpx"
)

// Persian user-facing messages (contracts/api.md: all user-facing text is Persian).
const (
	msgTitleRequired     = "عنوان درخواست الزامی است."
	msgTitleTooLong      = "عنوان درخواست نباید بیشتر از ۱۵۰ کاراکتر باشد."
	msgCategoryInvalid   = "دسته‌بندی درخواست نامعتبر است."
	msgPriorityInvalid   = "اولویت درخواست نامعتبر است."
	msgStatusInvalid     = "وضعیت درخواست نامعتبر است."
	msgTransitionInvalid = "تغییر وضعیت درخواست مجاز نیست."
	msgNotResident       = "برای ثبت درخواست باید ساکن یک واحد باشید."
	msgNotFound          = "درخواست یافت نشد."
	msgNoAccess          = "شما به این درخواست دسترسی ندارید."
	msgNoBuildingAccess  = "شما به این ساختمان دسترسی ندارید."
	msgAssigneeNotFound  = "شخص مسئول یافت نشد."
	msgCostInvalid       = "هزینه ثبت‌شده باید صفر یا بیشتر باشد."
	msgPhotoNotFound     = "فایل تصویر یافت نشد."
	msgDescriptionLong   = "توضیحات بیش از حد طولانی است."
)

// Service owns US7 business rules: validation, the state machine, scope
// checks, per-transition notifications (FR-033) and audit entries (FR-038).
type Service struct {
	repo  *Repository
	notif *notification.Service
	audit *audit.Service
}

// NewService returns a maintenance service.
func NewService(repo *Repository, notif *notification.Service, aud *audit.Service) *Service {
	return &Service{repo: repo, notif: notif, audit: aud}
}

// SubmitInput is the resident submit payload (contracts/api.md POST /me/maintenance-requests).
type SubmitInput struct {
	Title       string     `json:"title"`
	Category    string     `json:"category"`
	Description *string    `json:"description"`
	Location    *string    `json:"location"`
	PhotoFileID *uuid.UUID `json:"photo_file_id"`
	Priority    string     `json:"priority"`
}

// UpdateInput is the manager patch payload (PATCH /maintenance-requests/{id}).
// Pointer fields distinguish omitted vs provided; a nil status means no state move.
type UpdateInput struct {
	Status           *string    `json:"status"`
	AssigneePersonID *uuid.UUID `json:"assignee_person_id"`
	RecordedCost     *json.Number `json:"recorded_cost"`
	Notes            *string    `json:"notes"`
}

// Persian status labels for notification bodies.
var statusPersian = map[string]string{
	StatusNew:         "ثبت‌شده",
	StatusUnderReview: "در حال بررسی",
	StatusInProgress:  "در حال انجام",
	StatusDone:        "انجام‌شده",
	StatusClosed:      "بسته‌شده",
}

// Submit validates the resident payload, resolves the resident's active unit/building,
// and creates a new request (status = new). The in-app notification
// request_submitted is emitted for the submitter (FR-033). Audited as
// maintenance.create.
func (s *Service) Submit(ctx context.Context, user *auth.User, in SubmitInput) (*MaintenanceRequest, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, httpx.BadRequest(msgTitleRequired).WithDetail("title", "required")
	}
	if len([]rune(title)) > 150 {
		return nil, httpx.BadRequest(msgTitleTooLong).WithDetail("title", "max_length")
	}
	cat := strings.TrimSpace(in.Category)
	if cat == "" || !validCategories[cat] {
		return nil, httpx.BadRequest(msgCategoryInvalid).WithDetail("category", "invalid")
	}
	pri := strings.TrimSpace(in.Priority)
	if pri == "" {
		pri = PriNormal
	}
	if !validPriorities[pri] {
		return nil, httpx.BadRequest(msgPriorityInvalid).WithDetail("priority", "invalid")
	}
	if in.Description != nil && len([]rune(*in.Description)) > 2000 {
		return nil, httpx.BadRequest(msgDescriptionLong).WithDetail("description", "max_length")
	}
	var photoPath *string
	if in.PhotoFileID != nil {
		p, err := s.repo.PhotoPath(ctx, *in.PhotoFileID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, httpx.BadRequest(msgPhotoNotFound).WithDetail("photo_file_id", "not_found")
			}
			return nil, err
		}
		photoPath = &p
	}
	// Resolve resident unit/building via persons.phone → occupancies (active).
	buildingID, unitID, ok, err := s.repo.ResolveResidentUnit(ctx, user.Phone)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgNotResident)
	}
	now := time.Now()
	m := &MaintenanceRequest{
		ID:          uuid.New(),
		BuildingID:  buildingID,
		UnitID:      &unitID,
		SubmittedBy: user.ID,
		Title:       title,
		Category:    cat,
		Description: in.Description,
		Location:    in.Location,
		PhotoFile:   photoPath,
		Priority:    pri,
		Status:      StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	// Best-effort in-app notification: request submitted.
	if s.notif != nil {
		refType := "maintenance_request"
		refID := m.ID
		body := "درخواست «" + m.Title + "» با موفقیت ثبت شد."
		_, _ = s.notif.Create(ctx, m.SubmittedBy, "request_submitted", "درخواست جدید ثبت شد", body, &refType, &refID)
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, &user.ID, "maintenance.create", "maintenance_request", &m.ID, nil, m)
	}
	return m, nil
}

// ListForResident returns the resident's own requests (isolation: submitted_by = caller).
func (s *Service) ListForResident(ctx context.Context, user *auth.User, page, size int) ([]MaintenanceRequest, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return s.repo.ListForUser(ctx, user.ID, page, size)
}

// GetForResident returns one own request or 403 when not owned.
func (s *Service) GetForResident(ctx context.Context, user *auth.User, id uuid.UUID) (*MaintenanceRequest, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.NotFound(msgNotFound)
		}
		return nil, err
	}
	if m.SubmittedBy != user.ID {
		return nil, httpx.Forbidden(msgNoAccess)
	}
	return m, nil
}

// ListForManager returns one filtered page of a building's requests (manager scope 403 otherwise).
func (s *Service) ListForManager(ctx context.Context, manager *auth.User, buildingID uuid.UUID, f Filter) ([]MaintenanceRequest, int64, error) {
	ok, err := s.repo.IsManagerOf(ctx, manager.ID, buildingID)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, httpx.Forbidden(msgNoBuildingAccess)
	}
	if f.Status != "" && !validStatuses[f.Status] {
		return nil, 0, httpx.BadRequest(msgStatusInvalid).WithDetail("status", "invalid")
	}
	if f.Priority != "" && !validPriorities[f.Priority] {
		return nil, 0, httpx.BadRequest(msgPriorityInvalid).WithDetail("priority", "invalid")
	}
	if f.Category != "" && !validCategories[f.Category] {
		return nil, 0, httpx.BadRequest(msgCategoryInvalid).WithDetail("category", "invalid")
	}
	f.BuildingID = buildingID
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 || f.Size > 100 {
		f.Size = 20
	}
	return s.repo.ListForBuilding(ctx, f)
}

// GetForManager returns one request within the manager's permitted building.
func (s *Service) GetForManager(ctx context.Context, manager *auth.User, id uuid.UUID) (*MaintenanceRequest, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.NotFound(msgNotFound)
		}
		return nil, err
	}
	ok, err := s.repo.IsManagerOf(ctx, manager.ID, m.BuildingID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgNoBuildingAccess)
	}
	return m, nil
}

// Update applies the manager patch: status move (state machine with 409 on illegal),
// assignee, recorded_cost, notes. Per-transition notification + audit are emitted
// when the status actually changes (FR-033/FR-038). Returns the mutated row and
// the before snapshot for optional handler-level auditing.
func (s *Service) Update(ctx context.Context, manager *auth.User, id uuid.UUID, in UpdateInput) (*MaintenanceRequest, *MaintenanceRequest, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, httpx.NotFound(msgNotFound)
		}
		return nil, nil, err
	}
	ok, err := s.repo.IsManagerOf(ctx, manager.ID, m.BuildingID)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, httpx.Forbidden(msgNoBuildingAccess)
	}
	before := *m // snapshot for audit

	// --- status transition -------------------------------------------------
	statusChanged := false
	var prevStatus string
	if in.Status != nil {
		next := strings.TrimSpace(*in.Status)
		if next == "" || !validStatuses[next] {
			return nil, nil, httpx.BadRequest(msgStatusInvalid).WithDetail("status", "invalid")
		}
		if next != m.Status {
			if !canTransition(m.Status, next) {
				return nil, nil, httpx.Conflict(msgTransitionInvalid).WithDetail("status", "illegal_transition")
			}
			prevStatus = m.Status
			m.Status = next
			statusChanged = true
			if next == StatusClosed {
				now := time.Now()
				m.ClosedAt = &now
			}
		}
	}

	// --- assignee ----------------------------------------------------------
	if in.AssigneePersonID != nil {
		if *in.AssigneePersonID == uuid.Nil {
			// explicit clear
			m.AssigneePersonID = nil
		} else {
			exists, err := s.repo.PersonExists(ctx, *in.AssigneePersonID)
			if err != nil {
				return nil, nil, err
			}
			if !exists {
				return nil, nil, httpx.BadRequest(msgAssigneeNotFound).WithDetail("assignee_person_id", "not_found")
			}
			m.AssigneePersonID = in.AssigneePersonID
		}
	}

	// --- recorded cost -----------------------------------------------------
	if in.RecordedCost != nil {
		raw := strings.TrimSpace(in.RecordedCost.String())
		// Empty string (JSON null arrived as "") → clear.
		if raw == "" || raw == "null" {
			m.RecordedCost = nil
		} else {
			// Accept both "500000" and 500000 (json.Number normalizes).
			// Also allow Persian-digit string? normalize via strings replacer if needed,
			// but the API contract is Latin digits — keep strict.
			cost, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || cost < 0 {
				return nil, nil, httpx.BadRequest(msgCostInvalid).WithDetail("recorded_cost", "gte_0")
			}
			m.RecordedCost = &cost
		}
	}

	// --- notes -------------------------------------------------------------
	if in.Notes != nil {
		// Empty string clears notes; otherwise set.
		if strings.TrimSpace(*in.Notes) == "" {
			m.Notes = nil
		} else {
			m.Notes = in.Notes
		}
	}

	m.UpdatedAt = time.Now()
	if err := s.repo.Save(ctx, m); err != nil {
		return nil, nil, err
	}

	if statusChanged {
		if s.notif != nil {
			label := statusPersian[m.Status]
			if label == "" {
				label = m.Status
			}
			prevLabel := statusPersian[prevStatus]
			if prevLabel == "" {
				prevLabel = prevStatus
			}
			refType := "maintenance_request"
			refID := m.ID
			title := "وضعیت درخواست تغییر کرد"
			body := "وضعیت درخواست «" + m.Title + "» به «" + label + "» تغییر کرد."
			_, _ = s.notif.Create(ctx, m.SubmittedBy, "request_status_changed", title, body, &refType, &refID)
		}
		if s.audit != nil {
			_ = s.audit.Append(ctx, &manager.ID, "maintenance.status_changed", "maintenance_request", &m.ID, before, m)
		}
	}
	return m, &before, nil
}
