package notifications

import (
	"context"
	"fmt"
	"mime"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

// Hub раздаёт новые уведомления открытым SSE-подключениям пользователя.
// Работает в пределах процесса: при нескольких репликах нужен общий канал (Redis Pub/Sub или
// fanout-очередь RabbitMQ), иначе live-события увидят только клиенты той же реплики.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan Notification]struct{}
}

func NewHub() *Hub { return &Hub{subs: map[string]map[chan Notification]struct{}{}} }

func (h *Hub) Subscribe(user string) (<-chan Notification, func()) {
	ch := make(chan Notification, 8)
	h.mu.Lock()
	if h.subs[user] == nil {
		h.subs[user] = map[chan Notification]struct{}{}
	}
	h.subs[user][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs[user], ch)
		if len(h.subs[user]) == 0 {
			delete(h.subs, user)
		}
		h.mu.Unlock()
	}
}

// Publish не блокируется: медленный клиент пропускает событие и догрузит список запросом.
func (h *Hub) Publish(user string, n Notification) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[user] {
		select {
		case ch <- n:
		default:
		}
	}
}

// Mailer — отправка письма. nil в Service — письма выключены.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTPMailer — простой SMTP без авторизации (локально Mailpit, http://localhost:8025).
// Для внешнего SMTP добавьте smtp.PlainAuth и TLS.
type SMTPMailer struct{ Addr, From string }

func (m SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("недопустимые символы в заголовках письма")
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + mime.QEncoding.Encode("utf-8", subject) +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nDate: " + time.Now().Format(time.RFC1123Z) + "\r\n\r\n" + body
	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(m.Addr, nil, m.From, []string{to}, []byte(msg)) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
