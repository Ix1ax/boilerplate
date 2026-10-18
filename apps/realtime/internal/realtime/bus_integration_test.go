//go:build integration

package realtime

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"
)

// Две «реплики» realtime с общим RabbitMQ: сообщение, опубликованное на одной,
// доходит до подписчика другой. Без этого чат ломается при масштабировании.
func TestBusFansOutAcrossReplicas(t *testing.T) {
	url := os.Getenv("AMQP_URL")
	if url == "" {
		t.Fatal("Укажите AMQP_URL (make infra)")
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	hubA, hubB := NewHub(), NewHub()
	busA, busB := NewBus(url, hubA, log), NewBus(url, hubB, log)
	go busA.Run(ctx)
	go busB.Run(ctx)
	// Подписчик на реплике B — «клиент» с буфером, без настоящего сокета.
	sub := &client{send: make(chan []byte, 4), cancel: func() {}}
	hubB.subscribe(sub, "room:it")
	time.Sleep(1500 * time.Millisecond) // обе реплики успевают объявить очереди
	if err := busA.Publish(ctx, Message{Channel: "room:it", From: "alice", Data: json.RawMessage(`{"n":1}`), At: time.Now()}); err != nil {
		t.Fatal(err)
	}
	select {
	case frame := <-sub.send:
		var f struct {
			Type    string  `json:"type"`
			Payload Message `json:"payload"`
		}
		if json.Unmarshal(frame, &f) != nil || f.Type != "message" || f.Payload.From != "alice" {
			t.Fatalf("%s", frame)
		}
	case <-ctx.Done():
		t.Fatal("сообщение не дошло до другой реплики")
	}
}
