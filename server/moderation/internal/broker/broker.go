// Package broker — обмен событиями через RabbitMQ: публикация из outbox и потребление очереди.
// Доставка at-least-once: обработчики обязаны быть идемпотентными.
package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"example.com/hackathon/server/moderation/internal/platform"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Файл одинаков во всех сервисах, отличается только путь модуля: копируйте целиком.
const (
	exchange = "events"
	dlx      = "events.dlx"
)

// Event — строка outbox, ожидающая публикации.
type Event struct {
	ID, Type, UserID, RequestID string
	Payload                     []byte
	CreatedAt                   time.Time
}

// Outbox — источник неотправленных событий (nil — сервис только потребляет).
type Outbox interface {
	ClaimEvents(ctx context.Context, n int) ([]Event, error)
	MarkSent(ctx context.Context, id string) error
}

// Options — очередь сервиса и ключи событий, на которые он подписан.
type Options struct {
	Queue  string
	Keys   []string
	Outbox Outbox
}

// Message — полученное событие. Handler вернул ошибку — сообщение вернётся в очередь (до лимита, затем в DLQ).
type Message struct {
	ID, Type, UserID, RequestID string
	Data                        json.RawMessage
}
type Handler func(ctx context.Context, m Message) error

// Broker держит соединение с RabbitMQ: если задан Outbox — публикует из него события,
// и всегда обрабатывает собственную очередь сервиса. Соединение восстанавливается само (Run).
type Broker struct {
	url     string
	outbox  Outbox
	handler Handler
	log     *slog.Logger
	queue   string
	keys    []string
}

func New(url string, opts Options, handler Handler, log *slog.Logger) *Broker {
	return &Broker{url: url, outbox: opts.Outbox, handler: handler, log: log, queue: opts.Queue, keys: opts.Keys}
}

// Run держит соединение с брокером и переподключается с задержкой, пока ctx не отменён.
func (b *Broker) Run(ctx context.Context) {
	delay := time.Second
	for ctx.Err() == nil {
		started := time.Now()
		err := b.session(ctx)
		if ctx.Err() != nil {
			return
		}
		b.log.Warn("Соединение с RabbitMQ потеряно", "error", err)
		if time.Since(started) > time.Minute {
			delay = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		if delay < 30*time.Second {
			delay *= 2
		}
	}
}

func (b *Broker) topology(ch *amqp.Channel) error {
	for _, ex := range []string{exchange, dlx} {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return err
		}
	}
	// Quorum-очередь с лимитом доставок: «ядовитое» сообщение уходит в DLQ, а не крутится вечно.
	if _, err := ch.QueueDeclare(b.queue, true, false, false, false, amqp.Table{
		"x-queue-type": "quorum", "x-delivery-limit": 5,
		"x-dead-letter-exchange": dlx, "x-dead-letter-routing-key": b.queue,
	}); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(b.queue+".dlq", true, false, false, false, amqp.Table{"x-queue-type": "quorum"}); err != nil {
		return err
	}
	if err := ch.QueueBind(b.queue+".dlq", b.queue, dlx, false, nil); err != nil {
		return err
	}
	for _, key := range b.keys {
		if err := ch.QueueBind(b.queue, key, exchange, false, nil); err != nil {
			return err
		}
	}
	return nil
}

func (b *Broker) session(ctx context.Context) error {
	conn, err := amqp.Dial(b.url)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	if err = ch.Confirm(false); err != nil {
		return err
	}
	if err = b.topology(ch); err != nil {
		return err
	}
	if err = ch.Qos(1, 0, false); err != nil {
		return err
	}
	msgs, err := ch.Consume(b.queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	b.log.Info("RabbitMQ подключён", "queue", b.queue)
	closed := conn.NotifyClose(make(chan *amqp.Error, 1))
	relayCtx, stop := context.WithCancel(ctx)
	defer stop()
	relayErr := make(chan error, 1)
	if b.outbox != nil {
		go func() { relayErr <- b.relay(relayCtx, ch) }()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-closed:
			return fmt.Errorf("соединение закрыто: %v", e)
		case err = <-relayErr:
			return err
		case d, ok := <-msgs:
			if !ok {
				return errors.New("канал потребителя закрыт")
			}
			b.consume(ctx, d)
		}
	}
}

func (b *Broker) consume(ctx context.Context, d amqp.Delivery) {
	var env struct {
		ID        string          `json:"id"`
		Type      string          `json:"type"`
		UserID    string          `json:"user_id"`
		RequestID string          `json:"request_id"`
		Data      json.RawMessage `json:"data"`
	}
	if json.Unmarshal(d.Body, &env) != nil {
		_ = d.Nack(false, false) // нечитаемое сообщение сразу в DLQ
		return
	}
	hctx, cancel := context.WithTimeout(platform.WithRequestID(ctx, env.RequestID), 2*time.Minute)
	defer cancel()
	if err := b.handler(hctx, Message{ID: env.ID, Type: env.Type, UserID: env.UserID, RequestID: env.RequestID, Data: env.Data}); err != nil {
		b.log.Warn("Обработка события не удалась", "type", env.Type, "request_id", env.RequestID, "error", err)
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}

// relay переносит события из outbox в брокер с подтверждением публикации (publisher confirms).
func (b *Broker) relay(ctx context.Context, ch *amqp.Channel) error {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
		events, err := b.outbox.ClaimEvents(ctx, 20)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			b.log.Warn("Не удалось прочитать outbox", "error", err)
			continue
		}
		for _, e := range events {
			body, _ := json.Marshal(map[string]any{"id": e.ID, "type": e.Type, "user_id": e.UserID, "request_id": e.RequestID, "occurred_at": e.CreatedAt, "data": json.RawMessage(e.Payload)})
			dc, err := ch.PublishWithDeferredConfirmWithContext(ctx, exchange, e.Type, false, false, amqp.Publishing{
				ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: e.ID, CorrelationId: e.RequestID, Body: body,
			})
			if err != nil {
				return err
			}
			if ok, err := dc.WaitContext(ctx); err != nil || !ok {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("брокер не подтвердил публикацию: %v", err)
			}
			if err = b.outbox.MarkSent(ctx, e.ID); err != nil {
				b.log.Warn("Событие отправлено, но не отмечено", "id", e.ID, "error", err)
			}
		}
	}
}
