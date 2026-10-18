// Package notifications — внутренние уведомления пользователя. Сервис не знает предметных
// областей: другие сервисы публикуют событие notification.requested, а он сохраняет и отдаёт его.
package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("Уведомление не найдено")

type Notification struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Link      string     `json:"link"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at"`
}
type Page struct {
	Items  []Notification `json:"items"`
	Total  int            `json:"total"`
	Unread int            `json:"unread"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type Store interface {
	List(ctx context.Context, user string, limit, offset int) (Page, error)
	Insert(ctx context.Context, user, eventID string, n Notification) error
	MarkRead(ctx context.Context, user, id string) error
	MarkAllRead(ctx context.Context, user string) error
}

type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }

func (s *Service) List(ctx context.Context, user string, limit, offset int) (Page, error) {
	return s.store.List(ctx, user, limit, offset)
}
func (s *Service) MarkRead(ctx context.Context, user, id string) error {
	return s.store.MarkRead(ctx, user, id)
}
func (s *Service) MarkAllRead(ctx context.Context, user string) error {
	return s.store.MarkAllRead(ctx, user)
}

// HandleEvent принимает notification.requested. Невалидные данные отбрасываются (не повторяются):
// ошибка издателя не должна зацикливать очередь. Ссылка допускается только внутренняя.
func (s *Service) HandleEvent(ctx context.Context, eventID, eventType, userID string, data []byte) error {
	if eventType != "notification.requested" || userID == "" {
		return nil
	}
	var in struct{ Title, Body, Link string }
	if json.Unmarshal(data, &in) != nil {
		return nil
	}
	in.Title, in.Body = strings.TrimSpace(in.Title), strings.TrimSpace(in.Body)
	if in.Title == "" || utf8.RuneCountInString(in.Title) > 200 || utf8.RuneCountInString(in.Body) > 1000 {
		return nil
	}
	if !strings.HasPrefix(in.Link, "/") || strings.HasPrefix(in.Link, "//") {
		in.Link = ""
	}
	if _, err := uuid.Parse(eventID); err != nil {
		eventID = uuid.NewString()
	}
	return s.store.Insert(ctx, userID, eventID, Notification{ID: uuid.NewString(), Title: in.Title, Body: in.Body, Link: in.Link})
}
