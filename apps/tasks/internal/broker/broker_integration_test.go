//go:build integration

package broker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type memOutbox struct {
	mu      sync.Mutex
	pending []Event
	sent    []string
}

func (o *memOutbox) ClaimEvents(context.Context, int) ([]Event, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := o.pending
	o.pending = nil
	return out, nil
}
func (o *memOutbox) MarkSent(_ context.Context, id string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sent = append(o.sent, id)
	return nil
}

// Сквозная проверка: событие из outbox доходит до обработчика, первая ошибка обработчика
// приводит к повторной доставке, а отправленное событие помечается в outbox.
func TestOutboxToConsumerWithRetry(t *testing.T) {
	url := os.Getenv("AMQP_URL")
	if url == "" {
		t.Fatal("Укажите AMQP_URL для интеграционного теста (make infra)")
	}
	ev := Event{ID: uuid.NewString(), Type: "task.created", UserID: "u-test", Payload: []byte(`{"task_id":"t-test"}`), CreatedAt: time.Now()}
	out := &memOutbox{pending: []Event{ev}}
	var mu sync.Mutex
	attempts := 0
	done := make(chan Message, 1)
	handler := func(_ context.Context, m Message) error {
		if m.UserID != "u-test" {
			return nil // чужие сообщения из общего exchange игнорируем
		}
		mu.Lock()
		defer mu.Unlock()
		if attempts++; attempts == 1 {
			return errors.New("временный сбой")
		}
		done <- m
		return nil
	}
	// Своя очередь: не мешаем работающему сервису.
	b := New(url, Options{Queue: "tasks.moderation.test-" + uuid.NewString()[:8], Keys: []string{"task.created"}, Outbox: out}, handler, slog.New(slog.NewTextHandler(io.Discard, nil)))
	// Тестовые очереди удаляем, иначе они навсегда останутся привязанными к exchange и будут копить сообщения.
	t.Cleanup(func() {
		conn, err := amqp.Dial(url)
		if err != nil {
			return
		}
		defer conn.Close()
		if ch, err := conn.Channel(); err == nil {
			_, _ = ch.QueueDelete(b.queue, false, false, false)
			_, _ = ch.QueueDelete(b.queue+".dlq", false, false, false)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	go b.Run(ctx)
	select {
	case m := <-done:
		if m.Type != "task.created" || string(m.Data) != `{"task_id":"t-test"}` {
			t.Fatalf("%+v", m)
		}
	case <-ctx.Done():
		t.Fatal("событие не доставлено за 30 секунд")
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Fatalf("ожидалась повторная доставка после ошибки, попыток: %d", attempts)
	}
	out.mu.Lock()
	defer out.mu.Unlock()
	if len(out.sent) != 1 || out.sent[0] != ev.ID {
		t.Fatalf("событие не отмечено отправленным: %v", out.sent)
	}
}
