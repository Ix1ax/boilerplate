package copilot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/hackathon/server/events/internal/events"
	"example.com/hackathon/server/events/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Getter — жалоба с описанием цели (текст, автор, число открытых жалоб) из БД events.
type Getter interface {
	AdminReport(ctx context.Context, id string) (events.Report, error)
}

// Action — обычный админский маршрут, которым модератор сам применяет решение.
type Action struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Result — подсказка по жалобе. Флаги, приоритет и вердикт задают правила; модель только
// добавляет совет. Ничего не сохраняется и не меняется: решение принимает админ.
type Result struct {
	ReportID   string `json:"report_id"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	// Verdict: needs_review — правила или модель видят признаки нарушения; ok — признаков нет.
	Verdict  string  `json:"verdict"`
	Priority string  `json:"priority"` // high | normal
	Flags    []Flag  `json:"flags"`
	Advice   *Advice `json:"advice"`
	// Source показывает, откуда рекомендация: только правила или правила + модель.
	Source string `json:"source"`
	// Accept — скрыть или удалить цель (закрывает жалобы как принятые); Dismiss — отклонить жалобы.
	Accept  Action `json:"accept"`
	Dismiss Action `json:"dismiss"`
	// DecisionRequired всегда true: система советует, решает человек.
	DecisionRequired bool `json:"decision_required"`
}

// Service разбирает жалобу: правила → совет модели (если доступна) → рекомендация админу.
type Service struct {
	reports Getter
	advisor Advisor // nil — только правила
}

func New(g Getter, a Advisor) *Service { return &Service{reports: g, advisor: a} }

// acceptPaths — чем админ принимает жалобу на каждый тип цели (те же маршруты, что без подсказки).
var acceptPaths = map[string]string{
	"event": "/api/v1/admin/events/%s/unpublish", "comment": "/api/v1/admin/comments/%s/hide",
	"review": "/api/v1/admin/reviews/%s/hide", "story": "/api/v1/admin/stories/%s/delete",
	"direct_message": "/api/v1/admin/messages/%s/hide",
}

// Analyze — подсказка по жалобе. Недоступность модели не ошибка: остаётся результат правил.
func (s *Service) Analyze(ctx context.Context, adminID, id string) (Result, error) {
	r, err := s.reports.AdminReport(ctx, id)
	if err != nil {
		return Result{}, err
	}
	text := r.Target.Text
	if r.TargetType == "event" {
		text = r.Target.Title + "\n" + text
	}
	res := Result{
		ReportID: r.ID, TargetType: r.TargetType, TargetID: r.TargetID,
		Flags: Check(text), Priority: Priority(r.Reason, r.OpenOnTarget), Source: "rules", DecisionRequired: true,
		Accept:  Action{"POST", fmt.Sprintf(acceptPaths[r.TargetType], r.TargetID)},
		Dismiss: Action{"POST", "/api/v1/admin/reports/" + r.ID + "/dismiss"},
	}
	if r.OpenOnTarget >= manyReports {
		res.Flags = append(res.Flags, Flag{"many_reports", fmt.Sprintf("На эту цель %d открытых жалоб", r.OpenOnTarget)})
	}
	if s.advisor != nil {
		llmCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		// Сбой или занятость модели не должны ломать разбор: остаются правила.
		c := Case{TargetType: r.TargetType, Reason: r.Reason, Content: text, Details: r.Details, Flags: res.Flags}
		if advice, err := s.advisor.Advise(llmCtx, adminID, c); err == nil {
			res.Advice, res.Source = &advice, "rules+llm"
		}
	}
	res.Verdict = "ok"
	if len(res.Flags) > 0 || (res.Advice != nil && res.Advice.Risk == "high") {
		res.Verdict = "needs_review"
	}
	return res, nil
}

// Register добавляет POST /admin/reports/:id/analyze в группу, где уже проверена роль admin.
// В gateway маршрут ограничен лимитом rate:llm: он может вызвать модель.
func Register(admin *gin.RouterGroup, s *Service) {
	admin.POST("/reports/:id/analyze", func(c *gin.Context) {
		id := c.Param("id")
		if uuid.Validate(id) != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		res, err := s.Analyze(c.Request.Context(), platform.Owner(c), id)
		switch {
		case errors.Is(err, events.ErrNotFound):
			fail(c, 404, "not_found", "Жалоба не найдена")
		case err != nil:
			_ = c.Error(err)
			fail(c, 500, "internal_error", "Не удалось подготовить подсказку")
		default:
			c.JSON(200, res)
		}
	})
}

func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
