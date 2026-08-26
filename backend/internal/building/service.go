package building

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/auth"
	"hamsa/internal/platform/httpx"
)

// Persian validation messages (contracts/api.md — all user-facing text is
// Persian).
const (
	msgNameRequired    = "نام ساختمان الزامی است."
	msgPhoneInvalid    = "شماره موبایل معتبر نیست."
	msgBuildingNF      = "ساختمان یافت نشد."
	msgForbidden       = "دسترسی غیرمجاز است."
	msgUnitNF          = "واحد یافت نشد."
	msgNumberRequired  = "شماره واحد الزامی است."
	msgAreaPositive    = "متراژ واحد باید عددی بزرگ‌تر از صفر باشد."
	msgStatusInvalid   = "وضعیت واحد معتبر نیست."
	msgFloorNegative   = "شماره طبقه نمی‌تواند منفی باشد."
	msgDuplicateNumber = "شماره واحد در این ساختمان تکراری است."
	msgCountNegative   = "تعداد نمی‌تواند منفی باشد."
	msgBuiltYear       = "سال ساخت معتبر نیست."
)

var (
	phoneRe   = regexp.MustCompile(`^09\d{9}$`)
	validUnit = map[string]bool{
		UnitStatusActive: true, UnitStatusVacant: true,
		UnitStatusOccupied: true, UnitStatusInactive: true,
	}
)

// ErrNoManager is returned when the caller has no manager context.
var ErrNoManager = errors.New("no authenticated manager")

// Service holds US2 business rules (validation, duplicate-number check,
// manager-scope filter, creator auto-grant).
type Service struct {
	repo *Repository
}

// NewService returns a building/unit service.
func NewService(repo *Repository) *Service { return &Service{repo: repo} }

// BuildingInput is the create/update payload for a building.
type BuildingInput struct {
	Name           string  `json:"name"`
	Address        *string `json:"address"`
	BlockCount     *int    `json:"block_count"`
	FloorCount     *int    `json:"floor_count"`
	UnitCount      *int    `json:"unit_count"`
	BuiltYear      *int    `json:"built_year"`
	ManagerPhone   *string `json:"manager_phone"`
	EmergencyPhone *string `json:"emergency_phone"`
	Notes          *string `json:"notes"`
}

// CreateBuilding validates input, persists the building, and auto-grants the
// creating manager in user_buildings.
func (s *Service) CreateBuilding(ctx context.Context, manager *auth.User, in BuildingInput) (*Building, error) {
	b, err := s.applyBuildingInput(&Building{ID: uuid.New()}, &in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateBuilding(ctx, b); err != nil {
		return nil, err
	}
	if err := s.repo.GrantManager(ctx, manager.ID, b.ID); err != nil {
		return nil, err
	}
	return b, nil
}

// UpdateBuilding loads, authorizes against the manager scope, applies the
// input, and persists. Returns before/after snapshots for audit.
func (s *Service) UpdateBuilding(ctx context.Context, manager *auth.User, id uuid.UUID, in BuildingInput) (*Building, *Building, error) {
	before, err := s.authorizedBuilding(ctx, manager, id)
	if err != nil {
		return nil, nil, err
	}
	snapshot := *before
	after, err := s.applyBuildingInput(before, &in)
	if err != nil {
		return nil, nil, err
	}
	if err := s.repo.UpdateBuilding(ctx, after); err != nil {
		return nil, nil, err
	}
	return after, &snapshot, nil
}

// GetBuilding returns a permitted building detail.
func (s *Service) GetBuilding(ctx context.Context, manager *auth.User, id uuid.UUID) (*Building, error) {
	return s.authorizedBuilding(ctx, manager, id)
}

// ListBuildings returns the manager's permitted buildings.
func (s *Service) ListBuildings(ctx context.Context, manager *auth.User) ([]Building, error) {
	return s.repo.ListBuildingsForManager(ctx, manager.ID)
}

// DeleteBuilding soft-deletes a permitted building.
func (s *Service) DeleteBuilding(ctx context.Context, manager *auth.User, id uuid.UUID) error {
	if _, err := s.authorizedBuilding(ctx, manager, id); err != nil {
		return err
	}
	return s.repo.DeleteBuilding(ctx, id)
}

// applyBuildingInput validates and copies input fields onto b.
func (s *Service) applyBuildingInput(b *Building, in *BuildingInput) (*Building, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, httpx.BadRequest(msgNameRequired).WithDetail("name", "required")
	}
	if in.ManagerPhone != nil && *in.ManagerPhone != "" && !phoneRe.MatchString(*in.ManagerPhone) {
		return nil, httpx.BadRequest(msgPhoneInvalid).WithDetail("manager_phone", "mobile_format")
	}
	if in.EmergencyPhone != nil && *in.EmergencyPhone != "" && !phoneRe.MatchString(*in.EmergencyPhone) {
		return nil, httpx.BadRequest(msgPhoneInvalid).WithDetail("emergency_phone", "mobile_format")
	}
	for name, v := range map[string]*int{
		"block_count": in.BlockCount, "floor_count": in.FloorCount, "unit_count": in.UnitCount,
	} {
		if v != nil && *v < 0 {
			return nil, httpx.BadRequest(msgCountNegative).WithDetail(name, "min_0")
		}
	}
	if in.BuiltYear != nil && (*in.BuiltYear < 1250 || *in.BuiltYear > 1500) {
		return nil, httpx.BadRequest(msgBuiltYear).WithDetail("built_year", "jalali_year")
	}

	b.Name = in.Name
	b.Address = in.Address
	if in.BlockCount != nil {
		b.BlockCount = *in.BlockCount
	}
	if in.FloorCount != nil {
		b.FloorCount = *in.FloorCount
	}
	if in.UnitCount != nil {
		b.UnitCount = *in.UnitCount
	}
	b.BuiltYear = in.BuiltYear
	b.ManagerPhone = in.ManagerPhone
	b.EmergencyPhone = in.EmergencyPhone
	b.Notes = in.Notes
	return b, nil
}

// authorizedBuilding loads a building and verifies the manager scope
// (403 regardless of existence when not permitted).
func (s *Service) authorizedBuilding(ctx context.Context, manager *auth.User, id uuid.UUID) (*Building, error) {
	if manager == nil {
		return nil, ErrNoManager
	}
	b, err := s.repo.FindBuilding(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, httpx.NotFound(msgBuildingNF)
	}
	if err != nil {
		return nil, err
	}
	ok, err := s.repo.CanManagerAccess(ctx, manager.ID, b.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgForbidden)
	}
	return b, nil
}

// --- units --------------------------------------------------------------------

// UnitInput is the create/update payload for a unit.
type UnitInput struct {
	Number         string  `json:"number"`
	Block          *string `json:"block"`
	Floor          *int    `json:"floor"`
	AreaM2         *int    `json:"area_m2"`
	ParkingCount   *int    `json:"parking_count"`
	ParkingNumbers *string `json:"parking_numbers"`
	StorageCount   *int    `json:"storage_count"`
	StorageNumbers *string `json:"storage_numbers"`
	Status         *string `json:"status"`
	Notes          *string `json:"notes"`
}

// CreateUnit validates input against a permitted building and inserts the
// unit; duplicate live numbers map to 409 (FR-003).
func (s *Service) CreateUnit(ctx context.Context, manager *auth.User, buildingID uuid.UUID, in UnitInput) (*Unit, error) {
	if _, err := s.authorizedBuilding(ctx, manager, buildingID); err != nil {
		return nil, err
	}
	u, err := s.applyUnitInput(&Unit{ID: uuid.New(), BuildingID: buildingID}, &in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateUnit(ctx, u); err != nil {
		if errors.Is(err, ErrDuplicateUnitNumber) {
			return nil, httpx.Conflict(msgDuplicateNumber).WithDetail("number", "unique_per_building")
		}
		return nil, err
	}
	return u, nil
}

// UpdateUnit loads, authorizes via the parent building scope, applies input,
// persists. Returns before/after snapshots for audit.
func (s *Service) UpdateUnit(ctx context.Context, manager *auth.User, id uuid.UUID, in UnitInput) (*Unit, *Unit, error) {
	before, err := s.authorizedUnit(ctx, manager, id)
	if err != nil {
		return nil, nil, err
	}
	snapshot := *before
	after, err := s.applyUnitInput(before, &in)
	if err != nil {
		return nil, nil, err
	}
	if err := s.repo.UpdateUnit(ctx, after); err != nil {
		if errors.Is(err, ErrDuplicateUnitNumber) {
			return nil, nil, httpx.Conflict(msgDuplicateNumber).WithDetail("number", "unique_per_building")
		}
		return nil, nil, err
	}
	return after, &snapshot, nil
}

// GetUnit returns a permitted unit detail.
func (s *Service) GetUnit(ctx context.Context, manager *auth.User, id uuid.UUID) (*Unit, error) {
	return s.authorizedUnit(ctx, manager, id)
}

// ListUnits returns one filtered page of a building's units plus total.
func (s *Service) ListUnits(ctx context.Context, manager *auth.User, buildingID uuid.UUID, f UnitFilter) ([]Unit, int64, error) {
	if _, err := s.authorizedBuilding(ctx, manager, buildingID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListUnits(ctx, buildingID, f)
}

// DeleteUnit soft-deletes (archives) a permitted unit.
func (s *Service) DeleteUnit(ctx context.Context, manager *auth.User, id uuid.UUID) error {
	if _, err := s.authorizedUnit(ctx, manager, id); err != nil {
		return err
	}
	return s.repo.DeleteUnit(ctx, id)
}

// applyUnitInput validates and copies input fields onto u.
func (s *Service) applyUnitInput(u *Unit, in *UnitInput) (*Unit, error) {
	in.Number = strings.TrimSpace(in.Number)
	if in.Number == "" {
		return nil, httpx.BadRequest(msgNumberRequired).WithDetail("number", "required")
	}
	if in.AreaM2 == nil && u.AreaM2 == 0 || in.AreaM2 != nil && *in.AreaM2 <= 0 {
		return nil, httpx.BadRequest(msgAreaPositive).WithDetail("area_m2", "gt_0")
	}
	if in.Floor != nil && *in.Floor < 0 {
		return nil, httpx.BadRequest(msgFloorNegative).WithDetail("floor", "min_0")
	}
	status := u.Status
	if in.Status != nil {
		if !validUnit[*in.Status] {
			return nil, httpx.BadRequest(msgStatusInvalid).WithDetail("status", "invalid_enum")
		}
		status = *in.Status
	}
	for name, v := range map[string]*int{
		"parking_count": in.ParkingCount, "storage_count": in.StorageCount,
	} {
		if v != nil && *v < 0 {
			return nil, httpx.BadRequest(msgCountNegative).WithDetail(name, "min_0")
		}
	}

	u.Number = in.Number
	u.Block = in.Block
	if in.Floor != nil {
		u.Floor = *in.Floor
	}
	if in.AreaM2 != nil {
		u.AreaM2 = *in.AreaM2
	}
	if in.ParkingCount != nil {
		u.ParkingCount = *in.ParkingCount
	}
	u.ParkingNumbers = in.ParkingNumbers
	if in.StorageCount != nil {
		u.StorageCount = *in.StorageCount
	}
	u.StorageNumbers = in.StorageNumbers
	u.Status = status
	u.Notes = in.Notes
	return u, nil
}

// authorizedUnit loads a unit and verifies its parent building is inside the
// manager scope.
func (s *Service) authorizedUnit(ctx context.Context, manager *auth.User, id uuid.UUID) (*Unit, error) {
	if manager == nil {
		return nil, ErrNoManager
	}
	u, err := s.repo.FindUnit(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, httpx.NotFound(msgUnitNF)
	}
	if err != nil {
		return nil, err
	}
	ok, err := s.repo.CanManagerAccess(ctx, manager.ID, u.BuildingID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgForbidden)
	}
	return u, nil
}
