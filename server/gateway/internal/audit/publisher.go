// Package audit публикует в RabbitMQ записи о изменяющих запросах (событие audit.recorded).
// Сохраняет их сервис audit. Доставка «по возможности»: у gateway нет базы для outbox, поэтому
// при недоступном брокере или переполнении буфера запись теряется (счётчик в логе).
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Record — одна запись журнала: кто, какой маршрут (шаблон, без значений параметров) и с каким итогом.
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

// Recorder — то, что нужно middleware; в тестах подменяется фейком.
type Recorder interface{ Record(Record) }

// Publisher копит записи в буфере и публикует их в фоне (Run), не задерживая ответы пользователю.
type Publisher struct {
	url     string
	queue   chan Record
	dropped atomic.Int64
	log     *slog.Logger
}

func NewPublisher(url string, log *slog.Logger) *Publisher {
	return &Publisher{url: url, queue: make(chan Record, 1024), log: log}
}

// Record не блокирует запрос: при полном буфере запись отбрасывается.
func (p *Publisher) Record(r Record) {
	select {
	case p.queue <- r:
	default:
		if n := p.dropped.Add(1); n%100 == 1 {
			p.log.Warn("Буфер журнала аудита переполнен, записи теряются", "dropped", n)
		}
	}
}

// Run держит соединение и публикует записи, переподключаясь с задержкой.
func (p *Publisher) Run(ctx context.Context) {
	for ctx.Err() == nil {
		if err := p.session(ctx); err != nil && ctx.Err() == nil {
			p.log.Warn("Публикация аудита: соединение с RabbitMQ потеряно", "error", err)
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
			}
		}
	}
}

func (p *Publisher) session(ctx context.Context) error {
	conn, err := amqp.Dial(p.url)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	if err = ch.ExchangeDeclare("events", "topic", true, false, false, false, nil); err != nil {
		return err
	}
	closed := conn.NotifyClose(make(chan *amqp.Error, 1))
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-closed:
			return e
		case r := <-p.queue:
			id := uuid.NewString()
			body, _ := json.Marshal(map[string]any{"id": id, "type": "audit.recorded", "user_id": r.ActorID, "request_id": r.RequestID, "occurred_at": r.OccurredAt, "data": r})
			if err = ch.PublishWithContext(ctx, "events", "audit.recorded", false, false, amqp.Publishing{
				ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: id, CorrelationId: r.RequestID, Body: body,
			}); err != nil {
				p.Record(r) // вернуть в буфер и переподключиться
				return err
			}
		}
	}
}
