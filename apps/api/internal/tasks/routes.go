package tasks

import (
	"errors"
	"example.com/hackathon/apps/api/internal/auth"
	"example.com/hackathon/apps/api/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Register(group *gin.RouterGroup, s *Service) {
	group.GET("/tasks", func(c *gin.Context) {
		limit, offset, ok := platform.ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
		if !ok {
			fail(c, 400, "invalid_pagination", "limit должен быть от 1 до 100, offset — от 0 до 100000")
			return
		}
		page, err := s.List(c.Request.Context(), auth.Current(c).Subject, limit, offset)
		if err != nil {
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось загрузить задачи")
			return
		}
		c.JSON(200, page)
	})
	group.POST("/tasks", func(c *gin.Context) {
		var input struct {
			Title string `json:"title"`
		}
		if c.ShouldBindJSON(&input) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полем title")
			return
		}
		task, err := s.Create(c.Request.Context(), auth.Current(c).Subject, input.Title)
		if errors.Is(err, ErrTitle) {
			fail(c, 400, "validation_error", err.Error())
			return
		}
		if err != nil {
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось сохранить задачу")
			return
		}
		c.JSON(201, task)
	})
	group.PATCH("/tasks/:id", func(c *gin.Context) {
		if _, err := uuid.Parse(c.Param("id")); err != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		var input struct {
			Completed *bool `json:"completed"`
		}
		if c.ShouldBindJSON(&input) != nil || input.Completed == nil {
			fail(c, 400, "invalid_body", "Укажите поле completed")
			return
		}
		task, err := s.Update(c.Request.Context(), auth.Current(c).Subject, c.Param("id"), *input.Completed)
		if handleError(c, err) {
			return
		}
		c.JSON(200, task)
	})
	group.DELETE("/tasks/:id", func(c *gin.Context) {
		if _, err := uuid.Parse(c.Param("id")); err != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		if handleError(c, s.Delete(c.Request.Context(), auth.Current(c).Subject, c.Param("id"))) {
			return
		}
		c.Status(204)
	})
}
func handleError(c *gin.Context, err error) bool {
	if errors.Is(err, ErrNotFound) {
		fail(c, 404, "not_found", "Задача не найдена")
		return true
	}
	if err != nil {
		_ = c.Error(err)
		fail(c, 500, "internal_error", "Не удалось изменить задачу")
		return true
	}
	return false
}
func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
