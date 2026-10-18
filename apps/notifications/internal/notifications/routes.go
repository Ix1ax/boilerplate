package notifications

import (
	"errors"

	"example.com/hackathon/apps/notifications/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Register(group *gin.RouterGroup, s *Service) {
	group.GET("/notifications", func(c *gin.Context) {
		limit, offset, ok := platform.ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
		if !ok {
			fail(c, 400, "invalid_pagination", "limit должен быть от 1 до 100, offset — от 0 до 100000")
			return
		}
		page, err := s.List(c.Request.Context(), platform.Owner(c), limit, offset)
		if err != nil {
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось загрузить уведомления")
			return
		}
		c.JSON(200, page)
	})
	group.POST("/notifications/read-all", func(c *gin.Context) {
		if err := s.MarkAllRead(c.Request.Context(), platform.Owner(c)); err != nil {
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось отметить уведомления")
			return
		}
		c.Status(204)
	})
	group.POST("/notifications/:id/read", func(c *gin.Context) {
		if _, err := uuid.Parse(c.Param("id")); err != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		err := s.MarkRead(c.Request.Context(), platform.Owner(c), c.Param("id"))
		switch {
		case errors.Is(err, ErrNotFound):
			fail(c, 404, "not_found", "Уведомление не найдено")
		case err != nil:
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось отметить уведомление")
		default:
			c.Status(204)
		}
	})
}
func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
