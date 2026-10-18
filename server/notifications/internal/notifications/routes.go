package notifications

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"example.com/hackathon/server/notifications/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Register добавляет маршруты /notifications: список, поток SSE, отметки прочтения
// и подключение MAX (статус, ссылка привязки, отвязка).
func Register(group *gin.RouterGroup, s *Service) {
	// SSE: новые уведомления приходят сразу. Соединение живёт до 2 минут, затем EventSource
	// в браузере переподключается сам; комментарий-пинг держит прокси от закрытия по простою.
	group.GET("/notifications/stream", func(c *gin.Context) {
		if s.Hub() == nil {
			fail(c, 503, "stream_unavailable", "Поток уведомлений отключён")
			return
		}
		events, unsubscribe := s.Hub().Subscribe(platform.Owner(c))
		defer unsubscribe()
		rc := http.NewResponseController(c.Writer)
		_ = rc.SetWriteDeadline(time.Now().Add(3 * time.Minute))
		h := c.Writer.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("X-Accel-Buffering", "no")
		c.Status(200)
		_, _ = c.Writer.WriteString(": connected\n\n")
		c.Writer.Flush()
		ping := time.NewTicker(25 * time.Second)
		defer ping.Stop()
		deadline := time.After(2 * time.Minute)
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case <-deadline:
				return
			case <-ping.C:
				_, _ = c.Writer.WriteString(": ping\n\n")
			case n := <-events:
				b, _ := json.Marshal(n)
				_, _ = c.Writer.WriteString("event: notification\ndata: " + string(b) + "\n\n")
			}
			c.Writer.Flush()
		}
	})
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
	// Канал MAX: статус, одноразовая ссылка привязки и отвязка.
	group.GET("/notifications/max", func(c *gin.Context) {
		enabled, linked, err := s.MaxStatus(c.Request.Context(), platform.Owner(c))
		if err != nil {
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось проверить подключение MAX")
			return
		}
		c.JSON(200, gin.H{"enabled": enabled, "linked": linked})
	})
	group.POST("/notifications/max/link", func(c *gin.Context) {
		url, err := s.MaxLinkURL(c.Request.Context(), platform.Owner(c))
		if errors.Is(err, ErrMaxDisabled) {
			fail(c, 503, "max_disabled", err.Error())
			return
		}
		if err != nil {
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось создать ссылку")
			return
		}
		c.JSON(200, gin.H{"url": url})
	})
	group.DELETE("/notifications/max", func(c *gin.Context) {
		err := s.MaxUnlink(c.Request.Context(), platform.Owner(c))
		if errors.Is(err, ErrMaxDisabled) {
			fail(c, 503, "max_disabled", err.Error())
			return
		}
		if err != nil {
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось отключить MAX")
			return
		}
		c.Status(204)
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
