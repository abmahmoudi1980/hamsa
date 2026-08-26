package auth

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/platform/httpx"
)

// ContextKey is the gin context key holding the authenticated *User.
const ContextKey = "auth.user"

// Authenticate returns middleware that parses the Bearer token, validates the
// JWT, loads the user, and stores it in the gin context (research.md R8: the
// JWT carries identity only; role/scope are resolved per request from the DB).
func Authenticate(tokens *TokenService, users *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			httpx.WriteError(c, httpx.Unauthorized("احراز هویت لازم است."))
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, prefix))

		userID, err := tokens.ParseAccessToken(raw)
		if err != nil {
			httpx.WriteError(c, httpx.Unauthorized("توکن نامعتبر یا منقضی شده است."))
			return
		}

		user, err := users.FindByID(c.Request.Context(), userID)
		if err != nil || !user.IsActive {
			httpx.WriteError(c, httpx.Unauthorized("حساب کاربری نامعتبر است."))
			return
		}

		c.Set(ContextKey, user)
		c.Next()
	}
}

// CurrentUser returns the authenticated user, or nil when the request is
// unauthenticated.
func CurrentUser(c *gin.Context) *User {
	if v, ok := c.Get(ContextKey); ok {
		if u, ok := v.(*User); ok {
			return u
		}
	}
	return nil
}

// RequireRole returns middleware that rejects users whose role is not in the
// allowed set (403 FORBIDDEN, per contracts/api.md).
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil {
			httpx.WriteError(c, httpx.Unauthorized("احراز هویت لازم است."))
			return
		}
		if _, ok := allowed[u.Role]; !ok {
			httpx.WriteError(c, httpx.Forbidden("دسترسی غیرمجاز است."))
			return
		}
		c.Next()
	}
}

// ScopeResolver resolves the building/unit scope a user may access
// (manager → user_buildings; resident → occupancies). The underlying tables are
// created by the US2 (user_buildings) and US3 (occupancies) migrations, so
// these methods activate once those migrations land; no Phase 2 route calls
// them yet.
type ScopeResolver struct {
	db *gorm.DB
}

// NewScopeResolver returns a scope resolver backed by db.
func NewScopeResolver(db *gorm.DB) *ScopeResolver { return &ScopeResolver{db: db} }

// ManagerBuildingIDs returns the building ids granted to a manager.
func (s *ScopeResolver) ManagerBuildingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := s.db.WithContext(ctx).
		Table("user_buildings").
		Where("user_id = ?", userID).
		Pluck("building_id", &ids).Error
	return ids, err
}

// ResidentUnitIDs returns the units the resident currently occupies, resolved
// via persons.phone → occupancies (end_date IS NULL = active).
func (s *ScopeResolver) ResidentUnitIDs(ctx context.Context, phone string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := s.db.WithContext(ctx).Raw(
		`SELECT DISTINCT o.unit_id
		   FROM occupancies o
		   JOIN persons p ON p.id = o.person_id
		  WHERE p.phone = ? AND p.deleted_at IS NULL AND o.end_date IS NULL`,
		phone,
	).Scan(&ids).Error
	return ids, err
}

// CanManagerAccessBuilding reports whether the manager may access a building.
func (s *ScopeResolver) CanManagerAccessBuilding(ctx context.Context, userID, buildingID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("user_buildings").
		Where("user_id = ? AND building_id = ?", userID, buildingID).
		Count(&count).Error
	return count > 0, err
}

// CanResidentAccessUnit reports whether the resident may access a unit.
func (s *ScopeResolver) CanResidentAccessUnit(ctx context.Context, phone string, unitID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Raw(
		`SELECT COUNT(1)
		   FROM occupancies o
		   JOIN persons p ON p.id = o.person_id
		  WHERE p.phone = ? AND p.deleted_at IS NULL AND o.end_date IS NULL AND o.unit_id = ?`,
		phone, unitID,
	).Scan(&count).Error
	return count > 0, err
}
