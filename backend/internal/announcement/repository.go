package announcement

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository provides announcement persistence + targeting helpers.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a repository backed by db.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// DB exposes the underlying handle for transactional composition.
func (r *Repository) DB() *gorm.DB { return r.db }

// ResidentUnits returns the minimal UnitInfo projection for a resident's
// currently-active occupancies (joins persons.phone → occupancies → units).
// Shared by the dashboard aggregator (US9 T082) so the resident panel can
// reuse the announcement module's audience-targeting math.
func (r *Repository) ResidentUnits(ctx context.Context, phone string) ([]UnitInfo, error) {
	if phone == "" {
		return []UnitInfo{}, nil
	}
	type row struct {
		ID         uuid.UUID `gorm:"column:id"`
		BuildingID uuid.UUID `gorm:"column:building_id"`
		Block      *string   `gorm:"column:block"`
		Floor      int       `gorm:"column:floor"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("occupancies o").
		Select("un.id, un.building_id, un.block, un.floor").
		Joins("JOIN persons p ON p.id = o.person_id").
		Joins("JOIN units un ON un.id = o.unit_id").
		Where("p.phone = ?", phone).
		Where("p.deleted_at IS NULL").
		Where("un.deleted_at IS NULL").
		Where("o.end_date IS NULL").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]UnitInfo, 0, len(rows))
	for _, x := range rows {
		out = append(out, UnitInfo{
			ID:         x.ID,
			BuildingID: x.BuildingID,
			Block:      x.Block,
			Floor:      x.Floor,
		})
	}
	return out, nil
}

// VisibleAnnouncements returns every announcement the resident can see right
// now (audience match + publish/expire window) with the resident's read
// state hydrated. Exposed publicly so US9 (resident panel) and any other
// reader-side consumer can build a snapshot without juggling pagination.
func (r *Repository) VisibleAnnouncements(ctx context.Context, residentUnits []UnitInfo, now time.Time) ([]AnnouncementWithRead, error) {
	all, err := r.visibleAnnouncements(ctx, residentUnits, now)
	if err != nil {
		return nil, err
	}
	out := make([]AnnouncementWithRead, 0, len(all))
	for _, a := range all {
		out = append(out, AnnouncementWithRead{Announcement: a})
	}
	return out, nil
}

// IsManagerOf reports whether userID has a user_buildings grant for buildingID.
func (r *Repository) IsManagerOf(ctx context.Context, userID, buildingID uuid.UUID) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Table("user_buildings").
		Where("user_id = ? AND building_id = ?", userID, buildingID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// AttachmentPath resolves a files-registry id to its storage path.
func (r *Repository) AttachmentPath(ctx context.Context, fileID uuid.UUID) (string, error) {
	var row struct {
		Path string `gorm:"column:path"`
	}
	if err := r.db.WithContext(ctx).Table("files").Select("path").Where("id = ?", fileID).Take(&row).Error; err != nil {
		return "", err
	}
	return row.Path, nil
}

// Create inserts an announcement.
func (r *Repository) Create(ctx context.Context, a *Announcement) error {
	return r.db.WithContext(ctx).Create(a).Error
}

// Get returns one announcement by id or gorm.ErrRecordNotFound.
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Announcement, error) {
	var a Announcement
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Save persists mutations on an announcement.
func (r *Repository) Save(ctx context.Context, a *Announcement) error {
	return r.db.WithContext(ctx).Save(a).Error
}

// Delete hard-deletes an announcement (announcement_reads cascade).
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Announcement{}, "id = ?", id).Error
}

// ListForBuilding returns one page of a building's announcements (manager view),
// newest first, with total count.
func (r *Repository) ListForBuilding(ctx context.Context, buildingID uuid.UUID, page, size int) ([]Announcement, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	var total int64
	if err := r.db.WithContext(ctx).Model(&Announcement{}).Where("building_id = ?", buildingID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Announcement
	if err := r.db.WithContext(ctx).Where("building_id = ?", buildingID).
		Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []Announcement{}
	}
	return items, total, nil
}

// UnitInfo is a minimal resident unit projection used for targeting.
type UnitInfo struct {
	ID         uuid.UUID
	BuildingID uuid.UUID
	Block      *string
	Floor      int
}

// TargetUserIDs returns distinct user ids whose active occupancy matches the
// announcement's audience. An empty slice means no resident matches.
func (r *Repository) TargetUserIDs(ctx context.Context, a *Announcement) ([]uuid.UUID, error) {
	q := r.db.WithContext(ctx).Table("users u").
		Joins("JOIN persons p ON p.phone = u.phone").
		Joins("JOIN occupancies o ON o.person_id = p.id").
		Joins("JOIN units un ON un.id = o.unit_id").
		Where("o.end_date IS NULL").
		Where("un.building_id = ?", a.BuildingID).
		Where("un.deleted_at IS NULL").
		Where("p.deleted_at IS NULL").
		Where("u.deleted_at IS NULL").
		Where("u.is_active = TRUE")

	switch a.AudienceType {
	case AudienceAll:
		// no extra filter
	case AudienceBlock:
		if a.AudienceValue == nil {
			return []uuid.UUID{}, nil
		}
		q = q.Where("COALESCE(un.block, '') = ?", *a.AudienceValue)
	case AudienceFloor:
		if a.AudienceValue == nil {
			return []uuid.UUID{}, nil
		}
		q = q.Where("un.floor::text = ?", *a.AudienceValue)
	case AudienceUnit:
		if a.AudienceValue == nil {
			return []uuid.UUID{}, nil
		}
		uid, err := uuid.Parse(*a.AudienceValue)
		if err != nil {
			return []uuid.UUID{}, nil
		}
		q = q.Where("un.id = ?", uid)
	default:
		return []uuid.UUID{}, nil
	}

	var ids []uuid.UUID
	if err := q.Distinct("u.id").Pluck("u.id", &ids).Error; err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uuid.UUID{}
	}
	return ids, nil
}

// ListForResident returns the paginated announcements visible to a resident
// whose active units are residentUnits, within the publish/expire window at now.
func (r *Repository) ListForResident(ctx context.Context, residentUnits []UnitInfo, now time.Time, page, size int) ([]AnnouncementWithRead, int64, error) {
	if len(residentUnits) == 0 {
		return []AnnouncementWithRead{}, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	visible, err := r.visibleAnnouncements(ctx, residentUnits, now)
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(visible))
	start := (page - 1) * size
	if start >= len(visible) {
		return []AnnouncementWithRead{}, total, nil
	}
	end := start + size
	if end > len(visible) {
		end = len(visible)
	}
	pageItems := visible[start:end]

	// Note: read state is populated by the service after fetching user reads.
	out := make([]AnnouncementWithRead, len(pageItems))
	for i, a := range pageItems {
		out[i] = AnnouncementWithRead{Announcement: a}
	}
	return out, total, nil
}

// visibleAnnouncements returns all of the resident's buildings' announcements
// inside the publish/expire window at now whose audience matches the resident
// units, newest first. Audience matching is in-memory (announcements per
// building are small in P0). Shared by the paginated list and the unread
// count so the API page-size clamp can never truncate the count.
func (r *Repository) visibleAnnouncements(ctx context.Context, residentUnits []UnitInfo, now time.Time) ([]Announcement, error) {
	// Build building -> blocks/floors/unitIds maps for the query.
	buildings := map[uuid.UUID]bool{}
	blockByBuilding := map[uuid.UUID]map[string]bool{}
	floorByBuilding := map[uuid.UUID]map[string]bool{}
	unitIDs := map[uuid.UUID]bool{}
	for _, u := range residentUnits {
		buildings[u.BuildingID] = true
		if u.Block != nil && *u.Block != "" {
			if blockByBuilding[u.BuildingID] == nil {
				blockByBuilding[u.BuildingID] = map[string]bool{}
			}
			blockByBuilding[u.BuildingID][*u.Block] = true
		}
		fs := floorToString(u.Floor)
		if floorByBuilding[u.BuildingID] == nil {
			floorByBuilding[u.BuildingID] = map[string]bool{}
		}
		floorByBuilding[u.BuildingID][fs] = true
		unitIDs[u.ID] = true
	}

	var buildingIDs []uuid.UUID
	for id := range buildings {
		buildingIDs = append(buildingIDs, id)
	}

	// Fetch candidates in window for those buildings.
	var candidates []Announcement
	if err := r.db.WithContext(ctx).
		Where("building_id IN ?", buildingIDs).
		Where("(publish_at IS NULL OR publish_at <= ?)", now).
		Where("(expire_at IS NULL OR expire_at > ?)", now).
		Order("created_at DESC").
		Find(&candidates).Error; err != nil {
		return nil, err
	}

	visible := make([]Announcement, 0, len(candidates))
	for _, a := range candidates {
		if isVisible(a, residentUnits, blockByBuilding, floorByBuilding, unitIDs) {
			visible = append(visible, a)
		}
	}
	return visible, nil
}

// ListAllVisibleIDs returns ids of all announcements visible to the resident
// (used for unread counts).
func (r *Repository) ListAllVisibleIDs(ctx context.Context, residentUnits []UnitInfo, now time.Time) ([]uuid.UUID, error) {
	items, err := r.visibleAnnouncements(ctx, residentUnits, now)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids, nil
}

// IsVisible reports whether a single announcement is visible to the resident.
func (r *Repository) IsVisible(ctx context.Context, announcementID uuid.UUID, residentUnits []UnitInfo, now time.Time) (bool, error) {
	a, err := r.Get(ctx, announcementID)
	if err != nil {
		return false, err
	}
	if a.PublishAt != nil && a.PublishAt.After(now) {
		return false, nil
	}
	if a.ExpireAt != nil && !a.ExpireAt.After(now) {
		return false, nil
	}
	buildings := map[uuid.UUID]bool{}
	blockByBuilding := map[uuid.UUID]map[string]bool{}
	floorByBuilding := map[uuid.UUID]map[string]bool{}
	unitIDs := map[uuid.UUID]bool{}
	for _, u := range residentUnits {
		buildings[u.BuildingID] = true
		if u.Block != nil && *u.Block != "" {
			if blockByBuilding[u.BuildingID] == nil {
				blockByBuilding[u.BuildingID] = map[string]bool{}
			}
			blockByBuilding[u.BuildingID][*u.Block] = true
		}
		fs := floorToString(u.Floor)
		if floorByBuilding[u.BuildingID] == nil {
			floorByBuilding[u.BuildingID] = map[string]bool{}
		}
		floorByBuilding[u.BuildingID][fs] = true
		unitIDs[u.ID] = true
	}
	if !buildings[a.BuildingID] {
		return false, nil
	}
	return isVisible(*a, residentUnits, blockByBuilding, floorByBuilding, unitIDs), nil
}

// MarkRead inserts the read row (idempotent via ON CONFLICT DO NOTHING).
func (r *Repository) MarkRead(ctx context.Context, announcementID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO announcement_reads (announcement_id, user_id) VALUES (?, ?)
		 ON CONFLICT (announcement_id, user_id) DO NOTHING`,
		announcementID, userID,
	).Error
}

// ReadsForUser returns announcement_ids the user has marked read.
func (r *Repository) ReadsForUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	var rows []AnnouncementRead
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[uuid.UUID]bool, len(rows))
	for _, row := range rows {
		m[row.AnnouncementID] = true
	}
	return m, nil
}

// UnreadCountForResident counts visible announcements not yet read.
func (r *Repository) UnreadCountForResident(ctx context.Context, residentUnits []UnitInfo, userID uuid.UUID, now time.Time) (int64, error) {
	ids, err := r.ListAllVisibleIDs(ctx, residentUnits, now)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	readMap, err := r.ReadsForUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	var unread int64
	for _, id := range ids {
		if !readMap[id] {
			unread++
		}
	}
	return unread, nil
}

func isVisible(a Announcement, units []UnitInfo, blocks map[uuid.UUID]map[string]bool, floors map[uuid.UUID]map[string]bool, unitIDs map[uuid.UUID]bool) bool {
	switch a.AudienceType {
	case AudienceAll:
		return true
	case AudienceBlock:
		if a.AudienceValue == nil {
			return false
		}
		return blocks[a.BuildingID][*a.AudienceValue]
	case AudienceFloor:
		if a.AudienceValue == nil {
			return false
		}
		return floors[a.BuildingID][*a.AudienceValue]
	case AudienceUnit:
		if a.AudienceValue == nil {
			return false
		}
		uid, err := uuid.Parse(*a.AudienceValue)
		if err != nil {
			return false
		}
		return unitIDs[uid]
	default:
		return false
	}
}

func floorToString(f int) string {
	return fmt.Sprintf("%d", f)
}
