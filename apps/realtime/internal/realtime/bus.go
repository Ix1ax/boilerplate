package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Bus доставляет сообщения на все реплики realtime через RabbitMQ: у каждой реплики своя
// временная очередь (exclusive, auto-delete), привязанная к событию realtime.publish в exchange
// events. Тем же событием любой сервис может отправить сообщение в канал (например user:<id>).
type Bus struct {
	url string
	hub *Hub
	log *slog.Logger
	ch  chan Message
}

const busKey = "realtime.publish"

func NewBus(url string, hub *Hub, log *slog.Logger) *Bus {
	return &Bus{url: url, hub: hub, log: log, ch: make(chan Message, 256)}
}

// Publish ставит сообщение в очередь отправки; при переполнении возвращает ошибку клиенту.
func (b *Bus) Publish(_ context.Context, m Message) error {
	select {
	case b.ch <- m:
		return nil
	default:
		return errBusFull
	}
}

type busError string

func (e busError) Error() string { return string(e) }

const errBusFull = busError("очередь отправки переполнена")

// Run держит соединение с брокером и переподключается; пока брокера нет, сообщения
// доставляются только локально, чтобы чат на одной реплике продолжал работать.
func (b *Bus) Run(ctx context.Context) {
	for ctx.Err() == nil {
		if err := b.session(ctx); err != nil && ctx.Err() == nil {
			b.log.Warn("realtime: соединение с RabbitMQ потеряно", "error", err)
			deadline := time.After(3 * time.Second)
		drain:
			for {
				select {
				case <-ctx.Done():
					return
				case m := <-b.ch:
					b.hub.Deliver(m)
				case <-deadline:
					break drain
				}
			}
		}
	}
}

func (b *Bus) session(ctx context.Context) error {
	conn, err := amqp.Dial(b.url)
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
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return err
	}
	if err = ch.QueueBind(q.Name, busKey, "events", false, nil); err != nil {
		return err
	}
	msgs, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		return err
	}
	closed := conn.NotifyClose(make(chan *amqp.Error, 1))
	b.log.Info("realtime: RabbitMQ подключён", "queue", q.Name)
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-closed:
			return e
		case m := <-b.ch:
			body, _ := json.Marshal(map[string]any{"type": busKey, "data": m})
			if err = ch.PublishWithContext(ctx, "events", busKey, false, false, amqp.Publishing{ContentType: "application/json", Body: body}); err != nil {
				b.hub.Deliver(m) // не потерять сообщение для своей реплики
				return err
			}
		case d, ok := <-msgs:
			if !ok {
				return busError("канал потребителя закрыт")
			}
			var env struct {
				Data Message `json:"data"`
			}
			if json.Unmarshal(d.Body, &env) == nil && env.Data.Channel != "" {
				if env.Data.At.IsZero() {
					env.Data.At = time.Now().UTC()
				}
				b.hub.Deliver(env.Data)
			}
		}
	}
}
