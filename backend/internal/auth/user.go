package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role values for users.role (migration 0001 user_role enum).
const (
	RoleManager  = "manager"
	RoleResident = "resident"
)

// User is the authentication identity (migration 0001 `users`).
type User struct {
	ID        uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	Phone     string     `gorm:"column:phone"`
	Role      string     `gorm:"column:role"`
	Name      string     `gorm:"column:name"`
	FCMToken  *string    `gorm:"column:fcm_token"`
	IsActive  bool       `gorm:"column:is_active"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

func (User) TableName() string { return "users" }

// Repository provides user persistence.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a user repository backed by db.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// FindByID returns the user by id, or gorm.ErrRecordNotFound. Soft-deleted
// users are excluded automatically (DeletedAt).
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByPhone returns the user by mobile number, or gorm.ErrRecordNotFound.
func (r *Repository) FindByPhone(ctx context.Context, phone string) (*User, error) {
	var u User
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Create inserts a new user. The caller must set ID (and Role/IsActive) before
// calling; GORM auto-fills CreatedAt/UpdatedAt.
func (r *Repository) Create(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// CountActive returns how many non-deleted users exist. Zero means a fresh
// deployment: the next registration becomes the manager (first-user
// bootstrap).
func (r *Repository) CountActive(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&User{}).Count(&n).Error
	return n, err
}
