package tasks

import (
	"context"
	"errors"
	"example.com/hackathon/server/tasks/internal/platform"
	"github.com/google/uuid"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrTitle = errors.New("Название должно содержать от 1 до 200 символов")
var ErrNotFound = errors.New("Задача не найдена")

// Workflow статусов задачи: визард создаёт черновик, дальше — обычная работа и архивация.
// Архивация требует комментария — причина должна быть видна в истории.
var statusWorkflow = platform.NewWorkflow(map[string][]string{
	"draft":       {"todo"},
	"todo":        {"in_progress", "archived"},
	"in_progress": {"done", "todo"},
	"done":        {"archived", "in_progress"},
	"archived":    {"todo"},
}).RequireComment("archived")

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	// Moderation появляется после асинхронной ИИ-проверки (nil, пока она не выполнена).
	Moderation *Moderation `json:"moderation"`
}
type Moderation struct {
	Verdict   string    `json:"verdict"`
	Source    string    `json:"source"`
	CheckedAt time.Time `json:"checked_at"`
}
type Page struct {
	Items  []Task `json:"items"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}
type Store interface {
	List(context.Context, string, platform.ListQuery) (Page, error)
	Insert(context.Context, string, Task) error
	Get(context.Context, string, string) (Task, error)
	UpdateStatus(ctx context.Context, owner, id, to, comment, actor string, check func(from string) error) (Task, error)
	Delete(context.Context, string, string) error
	History(ctx context.Context, id string) ([]platform.Transition, error)
}
type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func (s *Service) List(ctx context.Context, owner string, q platform.ListQuery) (Page, error) {
	return s.store.List(ctx, owner, q)
}
func (s *Service) Create(ctx context.Context, owner, title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" || !utf8.ValidString(title) || utf8.RuneCountInString(title) > 200 {
		return Task{}, ErrTitle
	}
	task := Task{ID: uuid.NewString(), Title: title, Status: "draft", CreatedAt: time.Now().UTC()}
	return task, s.store.Insert(ctx, owner, task)
}
func (s *Service) Get(ctx context.Context, owner, id string) (Task, error) {
	return s.store.Get(ctx, owner, id)
}

// Update переводит задачу в новый статус: допустимость перехода и обязательность
// комментария проверяет statusWorkflow внутри одной транзакции с UPDATE (см. Repository.UpdateStatus).
func (s *Service) Update(ctx context.Context, owner, id, status, comment string) (Task, error) {
	actor := platform.Email(ctx)
	return s.store.UpdateStatus(ctx, owner, id, status, comment, actor, func(from string) error {
		return statusWorkflow.Check(from, status, comment)
	})
}
func (s *Service) Delete(ctx context.Context, owner, id string) error {
	return s.store.Delete(ctx, owner, id)
}

// History — лента переходов статуса; доступна только владельцу задачи.
func (s *Service) History(ctx context.Context, owner, id string) ([]platform.Transition, error) {
	if _, err := s.store.Get(ctx, owner, id); err != nil {
		return nil, err
	}
	return s.store.History(ctx, id)
}
