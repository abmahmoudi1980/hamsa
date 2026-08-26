package notification

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"hamsa/internal/auth"
	"hamsa/internal/platform/httpx"
)

// Register mounts the authenticated notification routes under r.
func Register(r *gin.RouterGroup, svc *Service) {
	r.GET("", list(svc))
	r.POST("/read-all", readAll(svc))
	r.POST("/:id/read", markRead(svc))
}

func list(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := auth.CurrentUser(c)
		if u == nil {
			httpx.WriteError(c, httpx.Unauthorized("احراز هویت لازم است."))
			return
		}

		unreadOnly := c.Query("unreadOnly") == "true"
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

		items, total, err := svc.List(c.Request.Context(), u.ID, unreadOnly, page, pageSize)
		if err != nil {
			httpx.WriteError(c, httpx.Internal("خطا در دریافت اعلان‌ها."))
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items":     items,
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		})
	}
}

func markRead(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := auth.CurrentUser(c)
		if u == nil {
			httpx.WriteError(c, httpx.Unauthorized("احراز هویت لازم است."))
			return
		}

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			httpx.WriteError(c, httpx.BadRequest("شناسه اعلان نامعتبر است."))
			return
		}

		if err := svc.MarkRead(c.Request.Context(), u.ID, id); err != nil {
			httpx.WriteError(c, httpx.Internal("خطا در ثبت وضعیت اعلان."))
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func readAll(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := auth.CurrentUser(c)
		if u == nil {
			httpx.WriteError(c, httpx.Unauthorized("احراز هویت لازم است."))
			return
		}

		if err := svc.MarkAllRead(c.Request.Context(), u.ID); err != nil {
			httpx.WriteError(c, httpx.Internal("خطا در ثبت وضعیت اعلان‌ها."))
			return
		}
		c.Status(http.StatusNoContent)
	}
}
