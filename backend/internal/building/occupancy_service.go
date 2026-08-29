package building

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/auth"
	"hamsa/internal/platform/httpx"
)

// Persian validation messages for US3 (contracts/api.md — all user-facing
// text is Persian).
const (
	msgPersonNameRequired = "نام شخص الزامی است."
	msgNationalIDInvalid  = "کد ملی معتبر نیست."
	msgPersonNotFound     = "شخص یافت نشد."
	msgRelInvalid         = "نسبت ساکن معتبر نیست."
	msgOccDatesInvalid    = "تاریخ پایان نمی‌تواند قبل از تاریخ شروع باشد."
	msgDateRequired       = "تاریخ الزامی است."
	msgOccNotFound        = "سابقه سکونت یافت نشد."
	msgOccAlreadyEnded    = "این سابقه سکونت قبلاً پایان یافته است."
	msgOccConflict        = "تغییر ساکن با وضعیت فعلی واحد هم‌خوانی ندارد."
	msgCountInvalid       = "تعداد ساکن نمی‌تواند منفی باشد."
)

var validRelationship = map[string]bool{
	RelOwner:            true,
	RelTenant:           true,
	RelNonResidentOwner: true,
}

// nationalIDValid checks the Iranian 10-digit national ID checksum
// (data-model.md "persons.national_id"): reject all-identical digits, then
// Σ digitᵢ×(10−i) mod 11 must equal the check digit (or 11−check when ≥ 2).
func nationalIDValid(id string) bool {
	if len(id) != 10 {
		return false
	}
	same := true
	for _, c := range id[1:] {
		if c != rune(id[0]) {
			same = false
			break
		}
	}
	if same {
		return false
	}
	sum := 0
	for i := range 9 {
		d := int(id[i] - '0')
		if d < 0 || d > 9 {
			return false
		}
		sum += d * (10 - i)
	}
	last := int(id[9] - '0')
	if last < 0 || last > 9 {
		return false
	}
	r := sum % 11
	return (r < 2 && last == r) || (r >= 2 && last == 11-r)
}

// PersonInput is the create/update payload for a person.
type PersonInput struct {
	FullName   string  `json:"full_name"`
	Phone      *string `json:"phone"`
	NationalID *string `json:"national_id"`
}

// applyPersonInput validates and copies input fields onto p.
func (s *Service) applyPersonInput(p *Person, in *PersonInput) (*Person, error) {
	in.FullName = strings.TrimSpace(in.FullName)
	if in.FullName == "" {
		return nil, httpx.BadRequest(msgPersonNameRequired).WithDetail("full_name", "required")
	}
	if in.Phone != nil && *in.Phone != "" && !phoneRe.MatchString(*in.Phone) {
		return nil, httpx.BadRequest(msgPhoneInvalid).WithDetail("phone", "mobile_format")
	}
	if in.NationalID != nil {
		*in.NationalID = strings.TrimSpace(*in.NationalID)
		if *in.NationalID != "" && !nationalIDValid(*in.NationalID) {
			return nil, httpx.BadRequest(msgNationalIDInvalid).WithDetail("national_id", "checksum")
		}
		if *in.NationalID == "" {
			in.NationalID = nil
		}
	}
	p.FullName = in.FullName
	p.Phone = in.Phone
	p.NationalID = in.NationalID
	return p, nil
}

// CreatePerson validates input against a permitted building and inserts the
// person.
func (s *Service) CreatePerson(ctx context.Context, manager *auth.User, buildingID uuid.UUID, in PersonInput) (*Person, error) {
	if _, err := s.authorizedBuilding(ctx, manager, buildingID); err != nil {
		return nil, err
	}
	p, err := s.applyPersonInput(&Person{ID: uuid.New(), BuildingID: buildingID}, &in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreatePerson(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// authorizedPerson loads a person and verifies its building is in scope.
func (s *Service) authorizedPerson(ctx context.Context, manager *auth.User, id uuid.UUID) (*Person, error) {
	if manager == nil {
		return nil, ErrNoManager
	}
	p, err := s.repo.FindPerson(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, httpx.NotFound(msgPersonNotFound)
	}
	if err != nil {
		return nil, err
	}
	ok, err := s.repo.CanManagerAccess(ctx, manager.ID, p.BuildingID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgForbidden)
	}
	return p, nil
}

// UpdatePerson loads, authorizes, applies input, and persists. Returns
// before/after snapshots for audit.
func (s *Service) UpdatePerson(ctx context.Context, manager *auth.User, id uuid.UUID, in PersonInput) (*Person, *Person, error) {
	before, err := s.authorizedPerson(ctx, manager, id)
	if err != nil {
		return nil, nil, err
	}
	snapshot := *before
	after, err := s.applyPersonInput(before, &in)
	if err != nil {
		return nil, nil, err
	}
	if err := s.repo.UpdatePerson(ctx, after); err != nil {
		return nil, nil, err
	}
	return after, &snapshot, nil
}

// ListPersons returns one filtered page of a building's persons plus total.
func (s *Service) ListPersons(ctx context.Context, manager *auth.User, buildingID uuid.UUID, q string, page, size int) ([]Person, int64, error) {
	if _, err := s.authorizedBuilding(ctx, manager, buildingID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListPersons(ctx, buildingID, q, page, size)
}

// DeletePerson soft-deletes (archives) a permitted person; occupancy history
// is retained (spec §21).
func (s *Service) DeletePerson(ctx context.Context, manager *auth.User, id uuid.UUID) error {
	if _, err := s.authorizedPerson(ctx, manager, id); err != nil {
		return err
	}
	return s.repo.DeletePerson(ctx, id)
}

// --- occupancies ---------------------------------------------------------------

// OccupancyInput is the add-occupancy payload (contracts: person, relationship,
// start date; optional end date).
type OccupancyInput struct {
	PersonID     uuid.UUID `json:"person_id"`
	Relationship string    `json:"relationship"`
	StartDate    Date      `json:"start_date"`
	EndDate      *Date     `json:"end_date"`
}

// AddOccupancy links a person to a unit (FR-005/FR-006). Adding a new active
// occupancy with the same relationship closes the previous one (FR-007,
// one active tenant per unit enforced in the repository transaction).
func (s *Service) AddOccupancy(ctx context.Context, manager *auth.User, unitID uuid.UUID, in OccupancyInput) (*Occupancy, error) {
	u, err := s.authorizedUnit(ctx, manager, unitID)
	if err != nil {
		return nil, err
	}
	if !validRelationship[in.Relationship] {
		return nil, httpx.BadRequest(msgRelInvalid).WithDetail("relationship", "invalid_enum")
	}
	if in.StartDate.IsZero() {
		return nil, httpx.BadRequest(msgDateRequired).WithDetail("start_date", "required")
	}
	if in.EndDate != nil && in.EndDate.Before(in.StartDate.Time) {
		return nil, httpx.BadRequest(msgOccDatesInvalid).WithDetail("end_date", "gte_start_date")
	}
	p, err := s.authorizedPerson(ctx, manager, in.PersonID)
	if err != nil {
		return nil, err
	}
	if p.BuildingID != u.BuildingID {
		return nil, httpx.BadRequest(msgRelInvalid).WithDetail("person_id", "wrong_building")
	}
	o := &Occupancy{
		ID:           uuid.New(),
		UnitID:       unitID,
		PersonID:     in.PersonID,
		Relationship: in.Relationship,
		StartDate:    in.StartDate,
		EndDate:      in.EndDate,
		IsActive:     true,
	}
	if err := s.repo.CreateOccupancy(ctx, o); err != nil {
		if errors.Is(err, ErrOccupancyConflict) {
			return nil, httpx.Conflict(msgOccConflict).WithDetail("relationship", "one_active_per_unit")
		}
		return nil, err
	}
	return o, nil
}

// ListOccupancies returns the full dated history of a unit's occupancies,
// persons included (FR-007: history preserved).
func (s *Service) ListOccupancies(ctx context.Context, manager *auth.User, unitID uuid.UUID) ([]Occupancy, error) {
	if _, err := s.authorizedUnit(ctx, manager, unitID); err != nil {
		return nil, err
	}
	return s.repo.ListOccupancies(ctx, unitID)
}

// EndOccupancyInput is the PATCH payload for end-dating an occupancy.
type EndOccupancyInput struct {
	EndDate Date `json:"end_date"`
}

// EndOccupancy end-dates an occupancy row, preserving it (FR-007). The row
// itself is never deleted.
func (s *Service) EndOccupancy(ctx context.Context, manager *auth.User, id uuid.UUID, in EndOccupancyInput) (*Occupancy, error) {
	o, err := s.repo.FindOccupancy(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, httpx.NotFound(msgOccNotFound)
	}
	if err != nil {
		return nil, err
	}
	if _, err := s.authorizedUnit(ctx, manager, o.UnitID); err != nil {
		return nil, err
	}
	if o.EndDate != nil {
		return nil, httpx.Conflict(msgOccAlreadyEnded).WithDetail("end_date", "already_set")
	}
	if in.EndDate.IsZero() {
		return nil, httpx.BadRequest(msgDateRequired).WithDetail("end_date", "required")
	}
	if in.EndDate.Before(o.StartDate.Time) {
		return nil, httpx.BadRequest(msgOccDatesInvalid).WithDetail("end_date", "gte_start_date")
	}
	if err := s.repo.EndOccupancy(ctx, id, in.EndDate); err != nil {
		return nil, err
	}
	o.EndDate = &in.EndDate
	o.IsActive = false
	return o, nil
}

// --- occupant counts -----------------------------------------------------------

// OccupantCountInput is the record-occupant-count payload (FR-008).
type OccupantCountInput struct {
	OccupantCount int  `json:"occupant_count"`
	EffectiveFrom Date `json:"effective_from"`
}

// RecordOccupantCount appends one occupant-count data point (BR-04: counted
// independently of occupancies, with full history).
func (s *Service) RecordOccupantCount(ctx context.Context, manager *auth.User, unitID uuid.UUID, in OccupantCountInput) (*OccupantCount, error) {
	if _, err := s.authorizedUnit(ctx, manager, unitID); err != nil {
		return nil, err
	}
	if in.OccupantCount < 0 {
		return nil, httpx.BadRequest(msgCountInvalid).WithDetail("occupant_count", "min_0")
	}
	if in.EffectiveFrom.IsZero() {
		return nil, httpx.BadRequest(msgDateRequired).WithDetail("effective_from", "required")
	}
	oc := &OccupantCount{
		ID:            uuid.New(),
		UnitID:        unitID,
		OccupantCount: in.OccupantCount,
		EffectiveFrom: in.EffectiveFrom,
	}
	if err := s.repo.RecordOccupantCount(ctx, oc); err != nil {
		return nil, err
	}
	return oc, nil
}

// ListOccupantCounts returns the full occupant-count history of a unit,
// newest first.
func (s *Service) ListOccupantCounts(ctx context.Context, manager *auth.User, unitID uuid.UUID) ([]OccupantCount, error) {
	if _, err := s.authorizedUnit(ctx, manager, unitID); err != nil {
		return nil, err
	}
	return s.repo.ListOccupantCounts(ctx, unitID)
}

// OccupantCountAsOf exposes the as-of read used by the charge engine
// (data-model.md: max effective_from ≤ reference date).
func (s *Service) OccupantCountAsOf(ctx context.Context, unitID uuid.UUID, ref time.Time) (int, error) {
	return s.repo.OccupantCountAsOf(ctx, unitID, ref)
}
