package tasks

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrTitle = errors.New("Название должно содержать от 1 до 200 символов")
var ErrNotFound = errors.New("Задача не найдена")

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
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
	List(context.Context, string, int, int) (Page, error)
	Insert(context.Context, string, Task) error
	Get(context.Context, string, string) (Task, error)
	Update(context.Context, string, string, bool) (Task, error)
	Delete(context.Context, string, string) error
}
type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func (s *Service) List(ctx context.Context, owner string, limit, offset int) (Page, error) {
	return s.store.List(ctx, owner, limit, offset)
}
func (s *Service) Create(ctx context.Context, owner, title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" || !utf8.ValidString(title) || utf8.RuneCountInString(title) > 200 {
		return Task{}, ErrTitle
	}
	task := Task{ID: uuid.NewString(), Title: title, CreatedAt: time.Now().UTC()}
	return task, s.store.Insert(ctx, owner, task)
}
func (s *Service) Get(ctx context.Context, owner, id string) (Task, error) {
	return s.store.Get(ctx, owner, id)
}
func (s *Service) Update(ctx context.Context, owner, id string, completed bool) (Task, error) {
	return s.store.Update(ctx, owner, id, completed)
}
func (s *Service) Delete(ctx context.Context, owner, id string) error {
	return s.store.Delete(ctx, owner, id)
}
