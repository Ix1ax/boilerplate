package platform

import (
	"errors"
	"fmt"
)

// ErrTransition — недопустимый переход статуса (маршрут отвечает 409).
var ErrTransition = errors.New("Недопустимая смена статуса")

// Workflow — конечный автомат статусов: все разрешённые переходы описаны в одном месте.
// Предметная область задаёт только таблицу переходов, например:
//
//	var moderation = platform.NewWorkflow(map[string][]string{
//		"draft":     {"review"},
//		"review":    {"published", "rejected", "draft"}, // draft — возврат на доработку
//		"published": {"archived"},
//	})
type Workflow struct{ next map[string]map[string]bool }

func NewWorkflow(transitions map[string][]string) Workflow {
	w := Workflow{next: map[string]map[string]bool{}}
	for from, targets := range transitions {
		w.next[from] = map[string]bool{}
		for _, to := range targets {
			w.next[from][to] = true
		}
	}
	return w
}

// Check возвращает ErrTransition, если перейти из from в to нельзя.
func (w Workflow) Check(from, to string) error {
	if !w.next[from][to] {
		return fmt.Errorf("%w: %s → %s", ErrTransition, from, to)
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
