package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"example.com/hackathon/server/tasks/internal/platform"
	"example.com/hackathon/server/tasks/internal/tasks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Getter — чтение задачи с проверкой владельца в SQL.
type Getter interface {
	Get(ctx context.Context, owner, id string) (tasks.Task, error)
}

// Saver сохраняет результат проверки; nil — результат только возвращается.
type Saver interface {
	SaveModeration(ctx context.Context, owner, taskID string, m tasks.Moderation, result []byte) error
}

// Result — рекомендация по задаче. Флаги и вердикт задают правила; модель только добавляет совет.
type Result struct {
	TaskID  string  `json:"task_id"`
	Verdict string  `json:"verdict"` // ok | needs_review
	Flags   []Flag  `json:"flags"`
	Advice  *Advice `json:"advice"`
	// Source показывает, откуда рекомендация: только правила или правила + модель.
	Source string `json:"source"`
	// DecisionRequired всегда true: система советует, решает человек.
	DecisionRequired bool `json:"decision_required"`
}

// Service проверяет задачу: правила → совет модели (если доступна) → сохранение результата.
type Service struct {
	tasks   Getter
	advisor Advisor // nil — только правила
	saver   Saver   // nil — не сохранять
}

func New(t Getter, a Advisor, saver Saver) *Service {
	return &Service{tasks: t, advisor: a, saver: saver}
}

// HandleEvent — обработчик очереди: проверяет созданную задачу и сохраняет результат.
// Повторная доставка безопасна: результат записывается upsert-ом. Удалённая задача — не ошибка.
func (s *Service) HandleEvent(ctx context.Context, eventType, userID string, data []byte) error {
	if eventType != "task.created" {
		return nil
	}
	var payload struct {
		TaskID string `json:"task_id"`
		Email  string `json:"email"`
	}
	if json.Unmarshal(data, &payload) != nil || payload.TaskID == "" {
		return nil
	}
	ctx = platform.WithEmail(ctx, payload.Email)
	if _, err := s.Analyze(ctx, userID, payload.TaskID); err != nil && !errors.Is(err, tasks.ErrNotFound) {
		return err
	}
	return nil
}

// Analyze проверяет задачу владельца. Недоступность модели не ошибка: остаётся результат правил.
func (s *Service) Analyze(ctx context.Context, owner, id string) (Result, error) {
	task, err := s.tasks.Get(ctx, owner, id)
	if err != nil {
		return Result{}, err
	}
	res := Result{TaskID: task.ID, Flags: Check(task.Title), Source: "rules", DecisionRequired: true}
	if res.Flags == nil {
		res.Flags = []Flag{}
	}
	if s.advisor != nil {
		llmCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		// Сбой или занятость модели не должны ломать проверку: остаются правила.
		if advice, err := s.advisor.Advise(llmCtx, owner, task.Title, res.Flags); err == nil {
			res.Advice, res.Source = &advice, "rules+llm"
		}
	}
	res.Verdict = "ok"
	if len(res.Flags) > 0 || (res.Advice != nil && res.Advice.Risk == "high") {
		res.Verdict = "needs_review"
	}
	if s.saver != nil {
		raw, _ := json.Marshal(res)
		if err := s.saver.SaveModeration(ctx, owner, task.ID, tasks.Moderation{Verdict: res.Verdict, Source: res.Source}, raw); err != nil {
			return Result{}, err
		}
	}
	return res, nil
}

// Register добавляет POST /tasks/:id/analyze — ручную проверку (автоматическую запускает очередь).
func Register(group *gin.RouterGroup, s *Service) {
	group.POST("/tasks/:id/analyze", func(c *gin.Context) {
		id := c.Param("id")
		if _, err := uuid.Parse(id); err != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		res, err := s.Analyze(c.Request.Context(), platform.Owner(c), id)
		switch {
		case errors.Is(err, tasks.ErrNotFound):
			fail(c, 404, "not_found", "Задача не найдена")
		case err != nil:
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось проверить задачу")
		default:
			c.JSON(200, res)
		}
	})
}
func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
