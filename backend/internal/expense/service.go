package expense

// T065 — expense service: validation, manager-scope authorization, receipt
// binding via the T010 files registry, and the approval workflow.

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/auth"
	"hamsa/internal/building"
	"hamsa/internal/platform/httpx"
)

// Input is the create/update payload (contracts/api.md). Amount accepts a
// JSON number or string (json.Number); dates are ISO-8601 YYYY-MM-DD.
type Input struct {
	Title          string      `json:"title"`
	Category       string      `json:"category"`
	Amount         json.Number `json:"amount"`
	ExpenseDate    string      `json:"expense_date"`
	Description    *string     `json:"description"`
	PayerPersonID  *uuid.UUID  `json:"payer_person_id"`
	ReceiptFileID  *string     `json:"receipt_file_id"`
	ApprovalStatus string      `json:"approval_status"`
}

// Service implements the US6 business rules.
type Service struct {
	repo *Repository
}

// NewService returns an expense service.
func NewService(repo *Repository) *Service { return &Service{repo: repo} }

// Create validates the input and records a new expense (audited by the
// handler decorator). The manager must hold a user_buildings grant.
func (s *Service) Create(ctx context.Context, manager *auth.User, buildingID uuid.UUID, in Input) (*Expense, error) {
	if ok, err := s.repo.IsManagerOf(ctx, manager.ID, buildingID); err != nil {
		return nil, err
	} else if !ok {
		return nil, httpx.Forbidden("شما به این ساختمان دسترسی ندارید")
	}
	e := &Expense{ID: uuid.New(), BuildingID: buildingID, CreatedBy: &manager.ID}
	if err := s.applyInput(ctx, e, in, true); err != nil {
		return nil, err
	}
	if err := s.repo.CreateExpense(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// Update applies a partial update (provided fields only) and advances the
// approval workflow (pending → approved/rejected). Audited by the decorator.
func (s *Service) Update(ctx context.Context, manager *auth.User, id uuid.UUID, in Input) (*Expense, error) {
	e, err := s.repo.GetExpense(ctx, id)
	if err != nil {
		return nil, err
	}
	if ok, err := s.repo.IsManagerOf(ctx, manager.ID, e.BuildingID); err != nil {
		return nil, err
	} else if !ok {
		return nil, httpx.Forbidden("شما به این ساختمان دسترسی ندارید")
	}
	if err := s.applyInput(ctx, e, in, false); err != nil {
		return nil, err
	}
	if err := s.repo.SaveExpense(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// Get returns one non-deleted expense within the manager's scope.
func (s *Service) Get(ctx context.Context, manager *auth.User, id uuid.UUID) (*Expense, error) {
	e, err := s.repo.GetExpense(ctx, id)
	if err != nil {
		return nil, err
	}
	if ok, err := s.repo.IsManagerOf(ctx, manager.ID, e.BuildingID); err != nil {
		return nil, err
	} else if !ok {
		return nil, httpx.Forbidden("شما به این ساختمان دسترسی ندارید")
	}
	return e, nil
}

// List returns one filtered page of the building's expenses.
func (s *Service) List(ctx context.Context, manager *auth.User, buildingID uuid.UUID, f ExpenseFilter) ([]Expense, int64, error) {
	if ok, err := s.repo.IsManagerOf(ctx, manager.ID, buildingID); err != nil {
		return nil, 0, err
	} else if !ok {
		return nil, 0, httpx.Forbidden("شما به این ساختمان دسترسی ندارید")
	}
	f.BuildingID = buildingID
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 || f.Size > 100 {
		f.Size = 20
	}
	return s.repo.ListExpenses(ctx, f)
}

// Delete soft-deletes an expense — the row survives for history (spec §21).
func (s *Service) Delete(ctx context.Context, manager *auth.User, id uuid.UUID) error {
	e, err := s.repo.GetExpense(ctx, id)
	if err != nil {
		return err
	}
	if ok, err := s.repo.IsManagerOf(ctx, manager.ID, e.BuildingID); err != nil {
		return err
	} else if !ok {
		return httpx.Forbidden("شما به این ساختمان دسترسی ندارید")
	}
	return s.repo.SoftDeleteExpense(ctx, id)
}

// Report builds the FR-027 financial report for month YYYY-MM.
func (s *Service) Report(ctx context.Context, manager *auth.User, buildingID uuid.UUID, month string) (*FinancialReport, error) {
	if ok, err := s.repo.IsManagerOf(ctx, manager.ID, buildingID); err != nil {
		return nil, err
	} else if !ok {
		return nil, httpx.Forbidden("شما به این ساختمان دسترسی ندارید")
	}
	y, m, ok := parseMonth(month)
	if !ok {
		return nil, httpx.BadRequest("ماه گزارش نامعتبر است؛ قالب صحیح YYYY-MM است")
	}
	rep, err := s.repo.Report(ctx, buildingID, y, m)
	if err != nil {
		return nil, err
	}
	rep.Month = month
	return rep, nil
}

// applyInput validates and applies the payload onto e. create=true requires
// every mandatory field; update applies only provided fields.
func (s *Service) applyInput(ctx context.Context, e *Expense, in Input, create bool) error {
	var verr *httpx.AppError
	fail := func() { verr = httpx.BadRequest("داده‌های هزینه نامعتبر است") }

	if in.Title != "" || create {
		t := strings.TrimSpace(in.Title)
		if t == "" || len([]rune(t)) > 150 {
			fail()
		}
		e.Title = t
	}
	if in.Category != "" || create {
		if !validCategory(in.Category) {
			fail()
		}
		e.Category = in.Category
	}
	if in.Amount.String() != "" || create {
		amt, err := strconv.ParseInt(in.Amount.String(), 10, 64)
		if err != nil || amt <= 0 {
			fail()
		}
		e.Amount = amt
	}
	if in.ExpenseDate != "" || create {
		d, err := time.Parse("2006-01-02", in.ExpenseDate)
		if err != nil {
			fail()
		}
		e.ExpenseDate = building.Date{Time: d}
	}
	if in.ApprovalStatus != "" || create {
		st := in.ApprovalStatus
		if st == "" {
			st = ApprovalPending
		}
		if st != ApprovalPending && st != ApprovalApproved && st != ApprovalRejected {
			fail()
		}
		e.ApprovalStatus = st
	}
	if in.Description != nil {
		e.Description = in.Description
	}
	if in.PayerPersonID != nil {
		e.PayerPersonID = in.PayerPersonID
	}
	if in.ReceiptFileID != nil {
		if *in.ReceiptFileID == "" {
			// Explicit clear (update) — remove the bound receipt.
			e.ReceiptFile = nil
		} else {
			fileID, err := uuid.Parse(*in.ReceiptFileID)
			if err != nil {
				return httpx.BadRequest("فایل رسید یافت نشد")
			}
			path, err := s.repo.ReceiptPath(ctx, fileID)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return httpx.BadRequest("فایل رسید یافت نشد")
				}
				return err
			}
			e.ReceiptFile = &path
		}
	}
	if verr != nil {
		return verr
	}
	return nil
}

func validCategory(c string) bool {
	for _, v := range Categories {
		if v == c {
			return true
		}
	}
	return false
}

// parseMonth strictly parses YYYY-MM (month 01–12).
func parseMonth(s string) (int, int, bool) {
	if len(s) != 7 || s[4] != '-' {
		return 0, 0, false
	}
	y, err1 := strconv.Atoi(s[:4])
	m, err2 := strconv.Atoi(s[5:7])
	if err1 != nil || err2 != nil || m < 1 || m > 12 || y < 1 {
		return 0, 0, false
	}
	return y, m, true
}
