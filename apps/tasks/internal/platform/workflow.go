package platform

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrTransition — недопустимый переход статуса (маршрут отвечает 409).
	ErrTransition = errors.New("Недопустимая смена статуса")
	// ErrCommentRequired — переход требует комментария (например, возврат на доработку; 400).
	ErrCommentRequired = errors.New("Укажите комментарий к смене статуса")
)

// Workflow — конечный автомат статусов: все разрешённые переходы описаны в одном месте.
// Предметная область задаёт только таблицу переходов, например:
//
//	var moderation = platform.NewWorkflow(map[string][]string{
//		"draft":     {"review"},
//		"review":    {"published", "rejected", "draft"}, // draft — возврат на доработку
//		"published": {"archived"},
//	})
type Workflow struct {
	next            map[string]map[string]bool
	commentRequired map[string]bool
}

func NewWorkflow(transitions map[string][]string) Workflow {
	w := Workflow{next: map[string]map[string]bool{}, commentRequired: map[string]bool{}}
	for from, targets := range transitions {
		w.next[from] = map[string]bool{}
		for _, to := range targets {
			w.next[from][to] = true
		}
	}
	return w
}

// RequireComment требует непустой комментарий при переходе в перечисленные статусы
// (возврат на доработку, отклонение): причина должна быть видна автору.
func (w Workflow) RequireComment(states ...string) Workflow {
	for _, s := range states {
		w.commentRequired[s] = true
	}
	return w
}

// Check проверяет переход и наличие комментария, если он обязателен.
func (w Workflow) Check(from, to, comment string) error {
	if !w.next[from][to] {
		return fmt.Errorf("%w: %s → %s", ErrTransition, from, to)
	}
	if w.commentRequired[to] && strings.TrimSpace(comment) == "" {
		return ErrCommentRequired
	}
	return nil
}

// Allowed — список статусов, доступных из текущего (для кнопок в интерфейсе).
func (w Workflow) Allowed(from string) []string {
	out := []string{}
	for to := range w.next[from] {
		out = append(out, to)
	}
	return out
}
