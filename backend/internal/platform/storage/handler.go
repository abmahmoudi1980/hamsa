package storage

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"hamsa/internal/auth"
	"hamsa/internal/platform/httpx"
)

// File mirrors the `files` registry table (migration 0001).
type File struct {
	ID          uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	Path        string     `gorm:"column:path"`
	ContentType string     `gorm:"column:content_type"`
	SizeBytes   int64      `gorm:"column:size_bytes"`
	UploadedBy  *uuid.UUID `gorm:"column:uploaded_by"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

func (File) TableName() string { return "files" }

// Register mounts the multipart upload route under r (already authenticated).
func Register(r *gin.RouterGroup, svc *Service, db *gorm.DB) {
	r.POST("", uploadHandler(svc, db))
}

func uploadHandler(svc *Service, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header, err := c.FormFile("file")
		if err != nil {
			httpx.WriteError(c, httpx.BadRequest("فایل ارسال نشده است."))
			return
		}

		f, err := header.Open()
		if err != nil {
			httpx.WriteError(c, httpx.BadRequest("خطا در خواندن فایل."))
			return
		}
		defer f.Close()

		id, relPath, contentType, size, err := svc.Save(f)
		if err != nil {
			switch err {
			case ErrUnsupportedType:
				httpx.WriteError(c, httpx.BadRequest("فقط فایل تصویر یا PDF مجاز است."))
			case ErrTooLarge:
				httpx.WriteError(c, httpx.BadRequest("حجم فایل بیش از حد مجاز است (حداکثر ۵ مگابایت)."))
			default:
				httpx.WriteError(c, httpx.Internal("خطا در ذخیره فایل."))
			}
			return
		}

		var uploadedBy *uuid.UUID
		if u := auth.CurrentUser(c); u != nil {
			uploadedBy = &u.ID
		}

		record := File{
			ID:          id,
			Path:        relPath,
			ContentType: contentType,
			SizeBytes:   size,
			UploadedBy:  uploadedBy,
		}
		if err := db.Create(&record).Error; err != nil {
			httpx.WriteError(c, httpx.Internal("خطا در ذخیره فایل."))
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":  id,
			"url": "/files/" + id.String(),
		})
	}
}
