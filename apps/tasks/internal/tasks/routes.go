package tasks

import (
	"errors"
	"example.com/hackathon/apps/tasks/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var taskListSpec = platform.ListSpec{
	Filters:     map[string]string{"status": "t.status"},
	Sorts:       map[string]string{"created": "t.created_at", "title": "t.title", "status": "t.status"},
	DefaultSort: "-created",
	Search:      []string{"t.title"},
}

func Register(group *gin.RouterGroup, s *Service) {
	group.GET("/tasks", func(c *gin.Context) {
		q, err := platform.ParseList(c, taskListSpec, 2)
		if err != nil {
			fail(c, 400, "invalid_query", err.Error())
			return
		}
		page, err := s.List(c.Request.Context(), platform.Owner(c), q)
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
		task, err := s.Create(c.Request.Context(), platform.Owner(c), input.Title)
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
			Status  string `json:"status"`
			Comment string `json:"comment"`
		}
		if c.ShouldBindJSON(&input) != nil || input.Status == "" {
			fail(c, 400, "invalid_body", "Укажите поле status")
			return
		}
		task, err := s.Update(c.Request.Context(), platform.Owner(c), c.Param("id"), input.Status, input.Comment)
		if handleError(c, err) {
			return
		}
		c.JSON(200, task)
	})
	group.GET("/tasks/:id/history", func(c *gin.Context) {
		if _, err := uuid.Parse(c.Param("id")); err != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		items, err := s.History(c.Request.Context(), platform.Owner(c), c.Param("id"))
		if handleError(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	group.DELETE("/tasks/:id", func(c *gin.Context) {
		if _, err := uuid.Parse(c.Param("id")); err != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		if handleError(c, s.Delete(c.Request.Context(), platform.Owner(c), c.Param("id"))) {
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
	if errors.Is(err, platform.ErrTransition) {
		fail(c, 409, "invalid_transition", err.Error())
		return true
	}
	if errors.Is(err, platform.ErrCommentRequired) {
		fail(c, 400, "comment_required", err.Error())
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
