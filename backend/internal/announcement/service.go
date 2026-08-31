package announcement

import (
	"context"
	"fmt"
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
	msgUnauthorized = "برای این عملیات باید وارد شوید"
	msgForbidden    = "دسترسی غیرمجاز است"
	msgNotFound     = "اطلاعیه یافت نشد"
	msgTitleRequired = "عنوان اطلاعیه الزامی است"
	msgBodyRequired  = "متن اطلاعیه الزامی است"
	msgAudienceInvalid = "نوع مخاطب نامعتبر است"
	msgAudienceValueRequired = "مقدار مخاطب برای این نوع الزامی است"
	msgAudienceAllNoValue    = "برای مخاطب «همه» مقدار نباید ارسال شود"
	msgAudienceValueInvalid  = "مقدار مخاطب نامعتبر است"
	msgUnitNotFound          = "واحد مورد نظر در این ساختمان یافت نشد"
	msgExpireBeforePublish   = "تاریخ انقضا باید بعد از تاریخ انتشار باشد"
	msgPublishAtInvalid      = "تاریخ انتشار نامعتبر است"
	msgExpireAtInvalid       = "تاریخ انقضا نامعتبر است"
)

var validAudience = map[string]bool{
	AudienceAll:   true,
	AudienceBlock: true,
	AudienceFloor: true,
	AudienceUnit:  true,
}

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// Service owns US8 business rules: validation, manager-scope authorization,
// audience targeting, publish/expire window, read tracking, and per-publish
// notifications (FR-032/FR-033).
type Service struct {
	repo  *Repository
	notif *notification.Service
	audit *audit.Service
	clock Clock
}

// NewService returns an announcement service.
func NewService(repo *Repository, notif *notification.Service, aud *audit.Service) *Service {
	return &Service{repo: repo, notif: notif, audit: aud, clock: realClock{}}
}

// SetClock overrides the clock (used in tests).
func (s *Service) SetClock(c Clock) { s.clock = c }

// Input is the create/update payload (contracts/api.md).
type Input struct {
	Title          string  `json:"title"`
	Body           string  `json:"body"`
	AudienceType   string  `json:"audience_type"`
	AudienceValue  *string `json:"audience_value"`
	PublishAt      *string `json:"publish_at"`
	ExpireAt       *string `json:"expire_at"`
	AttachmentFileID *string `json:"attachment_file_id"`
}

// Create publishes a new announcement for the building. The manager must hold
// a user_buildings grant. Emits announcement_published notifications to the
// targeted residents (best-effort).
func (s *Service) Create(ctx context.Context, manager *auth.User, buildingID uuid.UUID, in Input) (*Announcement, error) {
	if manager == nil {
		return nil, httpx.Unauthorized(msgUnauthorized)
	}
	ok, err := s.repo.IsManagerOf(ctx, manager.ID, buildingID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgForbidden)
	}

	a := &Announcement{
		ID:         uuid.New(),
		BuildingID: buildingID,
		CreatedBy:  &manager.ID,
	}
	if err := s.applyInput(ctx, a, in, true); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}

	// Best-effort notifications to targeted residents — only for announcements
	// already inside the publish/expire window (a future publish_at must not
	// notify before the announcement is visible; P0 has no scheduler, so a
	// scheduled publish notifies nothing yet).
	if s.inWindow(a, s.clock.Now()) {
		s.notifyTargets(ctx, a)
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, &manager.ID, "announcement.publish", "announcement", &a.ID, nil, a)
	}

	return a, nil
}

// Update edits an existing announcement (manager scope 403 otherwise).
func (s *Service) Update(ctx context.Context, manager *auth.User, id uuid.UUID, in Input) (*Announcement, error) {
	if manager == nil {
		return nil, httpx.Unauthorized(msgUnauthorized)
	}
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.NotFound(msgNotFound)
		}
		return nil, err
	}
	ok, err := s.repo.IsManagerOf(ctx, manager.ID, a.BuildingID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgForbidden)
	}

	before := *a
	if err := s.applyInput(ctx, a, in, false); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, a); err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, &manager.ID, "announcement.update", "announcement", &a.ID, before, a)
	}
	return a, nil
}

// Delete removes an announcement.
func (s *Service) Delete(ctx context.Context, manager *auth.User, id uuid.UUID) error {
	if manager == nil {
		return httpx.Unauthorized(msgUnauthorized)
	}
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound(msgNotFound)
		}
		return err
	}
	ok, err := s.repo.IsManagerOf(ctx, manager.ID, a.BuildingID)
	if err != nil {
		return err
	}
	if !ok {
		return httpx.Forbidden(msgForbidden)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, &manager.ID, "announcement.delete", "announcement", &a.ID, a, nil)
	}
	return nil
}

// GetForManager returns a permitted announcement.
func (s *Service) GetForManager(ctx context.Context, manager *auth.User, id uuid.UUID) (*Announcement, error) {
	if manager == nil {
		return nil, httpx.Unauthorized(msgUnauthorized)
	}
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.NotFound(msgNotFound)
		}
		return nil, err
	}
	ok, err := s.repo.IsManagerOf(ctx, manager.ID, a.BuildingID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, httpx.Forbidden(msgForbidden)
	}
	return a, nil
}

// ListForManager returns one page of the building's announcements.
func (s *Service) ListForManager(ctx context.Context, manager *auth.User, buildingID uuid.UUID, page, size int) ([]Announcement, int64, error) {
	if manager == nil {
		return nil, 0, httpx.Unauthorized(msgUnauthorized)
	}
	ok, err := s.repo.IsManagerOf(ctx, manager.ID, buildingID)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, httpx.Forbidden(msgForbidden)
	}
	return s.repo.ListForBuilding(ctx, buildingID, page, size)
}

// ListForResident returns the resident's targeted announcements within the
// publish/expire window, with read state populated.
func (s *Service) ListForResident(ctx context.Context, resident *auth.User, page, size int) ([]AnnouncementWithRead, int64, error) {
	if resident == nil {
		return nil, 0, httpx.Unauthorized(msgUnauthorized)
	}
	units, err := s.residentUnits(ctx, resident.Phone)
	if err != nil {
		return nil, 0, err
	}
	if len(units) == 0 {
		return []AnnouncementWithRead{}, 0, nil
	}
	now := s.clock.Now()
	items, total, err := s.repo.ListForResident(ctx, units, now, page, size)
	if err != nil {
		return nil, 0, err
	}
	// Populate read flags.
	readMap, err := s.repo.ReadsForUser(ctx, resident.ID)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		if readMap[items[i].ID] {
			items[i].IsRead = true
		}
	}
	// For full detail, also load read_at if needed (left nil; count uses map).
	return items, total, nil
}

// GetForResident returns one visible announcement with read state, or 404/403.
func (s *Service) GetForResident(ctx context.Context, resident *auth.User, id uuid.UUID) (*AnnouncementWithRead, error) {
	if resident == nil {
		return nil, httpx.Unauthorized(msgUnauthorized)
	}
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.NotFound(msgNotFound)
		}
		return nil, err
	}
	units, err := s.residentUnits(ctx, resident.Phone)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	visible, err := s.repo.IsVisible(ctx, id, units, now)
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, httpx.NotFound(msgNotFound)
	}
	readMap, _ := s.repo.ReadsForUser(ctx, resident.ID)
	awr := &AnnouncementWithRead{Announcement: *a, IsRead: readMap[id]}
	return awr, nil
}

// MarkRead marks one visible announcement as read (idempotent).
func (s *Service) MarkRead(ctx context.Context, resident *auth.User, id uuid.UUID) error {
	if resident == nil {
		return httpx.Unauthorized(msgUnauthorized)
	}
	// Verify existence.
	if _, err := s.repo.Get(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound(msgNotFound)
		}
		return err
	}
	units, err := s.residentUnits(ctx, resident.Phone)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	visible, err := s.repo.IsVisible(ctx, id, units, now)
	if err != nil {
		return err
	}
	if !visible {
		return httpx.NotFound(msgNotFound)
	}
	return s.repo.MarkRead(ctx, id, resident.ID)
}

// UnreadCount returns the resident's unread announcement count.
func (s *Service) UnreadCount(ctx context.Context, resident *auth.User) (int64, error) {
	if resident == nil {
		return 0, httpx.Unauthorized(msgUnauthorized)
	}
	units, err := s.residentUnits(ctx, resident.Phone)
	if err != nil {
		return 0, err
	}
	if len(units) == 0 {
		return 0, nil
	}
	return s.repo.UnreadCountForResident(ctx, units, resident.ID, s.clock.Now())
}

// residentUnits fetches the active units for the resident phone.
func (s *Service) residentUnits(ctx context.Context, phone string) ([]UnitInfo, error) {
	var units []UnitInfo
	err := s.repo.DB().WithContext(ctx).Raw(
		`SELECT DISTINCT u.id, u.building_id, u.block, u.floor
		   FROM units u
		   JOIN occupancies o ON o.unit_id = u.id
		   JOIN persons p ON p.id = o.person_id
		  WHERE p.phone = ? AND p.deleted_at IS NULL AND o.end_date IS NULL AND u.deleted_at IS NULL`,
		phone,
	).Scan(&units).Error
	if err != nil {
		return nil, err
	}
	if units == nil {
		units = []UnitInfo{}
	}
	return units, nil
}

func (s *Service) applyInput(ctx context.Context, a *Announcement, in Input, create bool) error {
	// Title / body required on create; optional on update (if provided, must be non-empty).
	if create || in.Title != "" {
		if strings.TrimSpace(in.Title) == "" {
			return httpx.BadRequest(msgTitleRequired)
		}
		if len([]rune(in.Title)) > 200 {
			return httpx.BadRequest("عنوان نباید بیش از ۲۰۰ نویسه باشد")
		}
		a.Title = strings.TrimSpace(in.Title)
	}
	if create || in.Body != "" {
		if strings.TrimSpace(in.Body) == "" {
			return httpx.BadRequest(msgBodyRequired)
		}
		a.Body = in.Body
	}

	// Audience type/value.
	if create || in.AudienceType != "" {
		if !validAudience[in.AudienceType] {
			return httpx.BadRequest(msgAudienceInvalid)
		}
		a.AudienceType = in.AudienceType
		// Normalize audience_value: treat empty string as nil for 'all'.
		var val *string
		if in.AudienceValue != nil && strings.TrimSpace(*in.AudienceValue) != "" {
			v := strings.TrimSpace(*in.AudienceValue)
			val = &v
		}
		if a.AudienceType == AudienceAll {
			if val != nil {
				return httpx.BadRequest(msgAudienceAllNoValue)
			}
			a.AudienceValue = nil
		} else {
			if val == nil {
				return httpx.BadRequest(msgAudienceValueRequired)
			}
			// Per-type validation.
			switch a.AudienceType {
			case AudienceUnit:
				uid, err := uuid.Parse(*val)
				if err != nil {
					return httpx.BadRequest(msgAudienceValueInvalid).WithDetail("audience_value", "uuid")
				}
				// Verify the unit belongs to the announcement's building.
				var cnt int64
				if err := s.repo.DB().WithContext(ctx).Table("units").
					Where("id = ? AND building_id = ? AND deleted_at IS NULL", uid, a.BuildingID).Count(&cnt).Error; err != nil {
					return err
				}
				if cnt == 0 {
					return httpx.BadRequest(msgUnitNotFound)
				}
			case AudienceFloor:
				// Must parse as integer.
				if _, err := fmt.Sscanf(*val, "%d", new(int)); err != nil {
					// Try stricter int parse.
					var n int
					if _, err2 := fmt.Sscanf(strings.TrimSpace(*val), "%d", &n); err2 != nil {
						return httpx.BadRequest(msgAudienceValueInvalid).WithDetail("audience_value", "floor_integer")
					}
				}
				// Normalize to canonical integer string.
				var n int
				fmt.Sscanf(*val, "%d", &n)
				canon := fmt.Sprintf("%d", n)
				val = &canon
			case AudienceBlock:
				if len(*val) > 20 {
					return httpx.BadRequest("نام بلوک نباید بیش از ۲۰ نویسه باشد")
				}
			}
			a.AudienceValue = val
		}
	} else if in.AudienceValue != nil {
		// Audience type not changed but value provided — treat as audience update attempt without type.
		return httpx.BadRequest(msgAudienceInvalid)
	}

	// Publish / expire times (RFC3339).
	if in.PublishAt != nil {
		if strings.TrimSpace(*in.PublishAt) == "" {
			a.PublishAt = nil
		} else {
			t, err := time.Parse(time.RFC3339, *in.PublishAt)
			if err != nil {
				return httpx.BadRequest(msgPublishAtInvalid)
			}
			a.PublishAt = &t
		}
	} else if create {
		a.PublishAt = nil
	}
	if in.ExpireAt != nil {
		if strings.TrimSpace(*in.ExpireAt) == "" {
			a.ExpireAt = nil
		} else {
			t, err := time.Parse(time.RFC3339, *in.ExpireAt)
			if err != nil {
				return httpx.BadRequest(msgExpireAtInvalid)
			}
			a.ExpireAt = &t
		}
	} else if create {
		a.ExpireAt = nil
	}
	if a.PublishAt != nil && a.ExpireAt != nil && !a.ExpireAt.After(*a.PublishAt) {
		return httpx.BadRequest(msgExpireBeforePublish)
	}

	// Attachment binding via files registry (T010).
	if in.AttachmentFileID != nil {
		if strings.TrimSpace(*in.AttachmentFileID) == "" {
			a.AttachmentFile = nil
		} else {
			fid, err := uuid.Parse(strings.TrimSpace(*in.AttachmentFileID))
			if err != nil {
				return httpx.BadRequest("شناسه فایل پیوست نامعتبر است")
			}
			path, err := s.repo.AttachmentPath(ctx, fid)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return httpx.BadRequest("فایل پیوست یافت نشد")
				}
				return err
			}
			a.AttachmentFile = &path
		}
	} else if create {
		a.AttachmentFile = nil
	}

	return nil
}

// inWindow reports whether the announcement is visible at t (publish_at <= t
// and expire_at > t — matching the repository window filters).
func (s *Service) inWindow(a *Announcement, t time.Time) bool {
	if a.PublishAt != nil && a.PublishAt.After(t) {
		return false
	}
	if a.ExpireAt != nil && !a.ExpireAt.After(t) {
		return false
	}
	return true
}

func (s *Service) notifyTargets(ctx context.Context, a *Announcement) {
	if s.notif == nil {
		return
	}
	ids, err := s.repo.TargetUserIDs(ctx, a)
	if err != nil || len(ids) == 0 {
		return
	}
	title := fmt.Sprintf("اطلاعیه جدید: %s", a.Title)
	body := a.Body
	if len([]rune(body)) > 200 {
		body = string([]rune(body)[:200]) + "…"
	}
	refType := "announcement"
	for _, uid := range ids {
		// Best-effort per user; failures are logged inside notification.Service.Create.
		_, _ = s.notif.Create(ctx, uid, "announcement_published", title, body, &refType, &a.ID)
	}
}
