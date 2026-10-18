package llm

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"example.com/hackathon/apps/llm/internal/ollama"
	"github.com/gin-gonic/gin"
)

type input struct {
	Prompt      string   `json:"prompt"`
	System      string   `json:"system"`
	JSON        bool     `json:"json"`
	Temperature *float64 `json:"temperature"`
}

func (i input) request() ollama.Request {
	return ollama.Request{Prompt: i.Prompt, System: i.System, JSON: i.JSON, Temperature: i.Temperature}
}

// Register добавляет маршруты. timeout продлевает дедлайн записи: ответ модели идёт дольше обычного.
func Register(group *gin.RouterGroup, s *Service, timeout time.Duration) {
	group.GET("/llm/status", func(c *gin.Context) { c.JSON(200, s.Status(c.Request.Context())) })
	group.POST("/llm/generate", func(c *gin.Context) {
		var in input
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полем prompt")
			return
		}
		extend(c, timeout)
		text, err := s.Generate(c.Request.Context(), in.request())
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"text": text, "model": s.Model()})
	})
	// SSE: каждый фрагмент — data: {"t":"..."}, конец — event: done, сбой — event: error.
	group.POST("/llm/stream", func(c *gin.Context) {
		var in input
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полем prompt")
			return
		}
		extend(c, timeout)
		started := false
		start := func() {
			if started {
				return
			}
			started = true
			h := c.Writer.Header()
			h.Set("Content-Type", "text/event-stream")
			h.Set("Cache-Control", "no-cache")
			h.Set("X-Accel-Buffering", "no")
			c.Status(200)
		}
		err := s.Stream(c.Request.Context(), in.request(), func(chunk string) error {
			start()
			b, _ := json.Marshal(gin.H{"t": chunk})
			if _, err := c.Writer.WriteString("data: " + string(b) + "\n\n"); err != nil {
				return err
			}
			c.Writer.Flush()
			return nil
		})
		if !started {
			if handle(c, err) {
				return
			}
			start()
		}
		if err != nil {
			_ = c.Error(err)
			_, _ = c.Writer.WriteString("event: error\ndata: {\"message\":\"Генерация прервана\"}\n\n")
		} else {
			_, _ = c.Writer.WriteString("event: done\ndata: {}\n\n")
		}
		c.Writer.Flush()
	})
}

func extend(c *gin.Context, d time.Duration) {
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(d + 10*time.Second))
}
func handle(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrPrompt):
		fail(c, 400, "validation_error", err.Error())
	case errors.Is(err, ErrBusy):
		c.Header("Retry-After", "5")
		fail(c, 429, "model_busy", err.Error())
	case errors.Is(err, ollama.ErrUnavailable):
		_ = c.Error(err)
		fail(c, 503, "model_unavailable", "Модель недоступна. Запустите её командой make llm")
	default:
		_ = c.Error(err)
		fail(c, 500, "internal_error", "Не удалось получить ответ модели")
	}
	return true
}
func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
