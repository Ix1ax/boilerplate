// Package audit — журнал изменяющих действий. Записи публикует gateway (событие audit.recorded),
// сервис только сохраняет их и отдаёт администратору. Предметной логики здесь нет.
package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Record struct {
	ActorID    string    `json:"actor"`
	ActorEmail string    `json:"actor_email"`
	Method     string    `json:"method"`
	Route      string    `json:"route"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	RequestID  string    `json:"request_id"`
	IP         string    `json:"ip"`
	OccurredAt time.Time `json:"occurred_at"`
}
type Page struct {
	Items  []Record `json:"items"`
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}

type Store interface {
	Insert(ctx context.Context, eventID string, r Record) error
}

type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }

// HandleEvent сохраняет audit.recorded. Повтор события не создаёт дубль (event_id уникален),
// невалидная запись отбрасывается, чтобы не зацикливать очередь.
func (s *Service) HandleEvent(ctx context.Context, eventID, eventType string, data []byte) error {
	if eventType != "audit.recorded" {
		return nil
	}
	var r Record
	if json.Unmarshal(data, &r) != nil || r.ActorID == "" || r.Method == "" || r.OccurredAt.IsZero() {
		return nil
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return nil
	}
	return s.store.Insert(ctx, eventID, r)
}
