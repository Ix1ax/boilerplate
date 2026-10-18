package maxbot

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/hackathon/server/notifications/internal/notifications"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

// fakeMAX — минимальный сервер MAX: принимает сообщения и один раз отдаёт обновления.
type fakeMAX struct {
	mu      sync.Mutex
	sent    []string // "user_id:text"
	auth    string
	updates string
}

func (f *fakeMAX) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.URL.Path {
	case "/messages":
		f.auth = r.Header.Get("Authorization")
		var body struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.sent = append(f.sent, r.URL.Query().Get("user_id")+":"+body.Text)
		_, _ = w.Write([]byte(`{"message":{"body":{"mid":"m1","text":"ok"}}}`))
	case "/updates":
		out := f.updates
		f.updates = `{"updates":[],"marker":2}`
		_, _ = w.Write([]byte(out))
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeMAX) messages() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.sent...)
}

type fakeLinks struct {
	mu       sync.Mutex
	codes    map[string]string
	linked   map[int64]string
	unlinked []int64
}

func (l *fakeLinks) ConsumeLinkCode(_ context.Context, code string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	user, ok := l.codes[code]
	if !ok {
		return "", notifications.ErrLinkCode
	}
	delete(l.codes, code)
	return user, nil
}
func (l *fakeLinks) Link(_ context.Context, user string, id int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.linked[id] = user
	return nil
}
func (l *fakeLinks) UnlinkMax(_ context.Context, id int64) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.linked[id]
	delete(l.linked, id)
	l.unlinked = append(l.unlinked, id)
	return ok, nil
}

func setup(t *testing.T) (*Bot, *fakeMAX, *fakeLinks) {
	f := &fakeMAX{updates: `{"updates":[],"marker":1}`}
	srv := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(srv.Close)
	links := &fakeLinks{codes: map[string]string{"good": "user-1"}, linked: map[int64]string{}}
	bot, err := New("secret-token", "demo_bot", srv.URL, links, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return bot, f, links
}

func TestSendUsesTokenAndUser(t *testing.T) {
	bot, f, _ := setup(t)
	if err := bot.Send(context.Background(), 42, "Привет"); err != nil {
		t.Fatal(err)
	}
	if got := f.messages(); len(got) != 1 || got[0] != "42:Привет" || f.auth != "secret-token" {
		t.Fatalf("%v auth=%q", got, f.auth)
	}
	if bot.LinkURL("abc") != "https://max.ru/demo_bot/start/abc" {
		t.Fatal(bot.LinkURL("abc"))
	}
}

func TestLinkFlow(t *testing.T) {
	bot, f, links := setup(t)
	ctx := context.Background()
	// Повторное использование кода не должно привязать второй аккаунт.
	bot.Handle(ctx, model.Update{UpdateType: model.UpdateBotStarted, UserID: 77, Payload: "good"})
	bot.Handle(ctx, model.Update{UpdateType: model.UpdateBotStarted, UserID: 88, Payload: "good"})
	if links.linked[77] != "user-1" || links.linked[88] != "" {
		t.Fatalf("%v", links.linked)
	}
	msgs := f.messages()
	if !strings.HasPrefix(msgs[0], "77:Готово") || !strings.Contains(msgs[1], "недействительна") {
		t.Fatalf("%v", msgs)
	}
	bot.Handle(ctx, model.Update{UpdateType: model.UpdateMessageCreated, UserID: 77, Message: &model.MessageUpdate{Body: model.MessageBody{Text: "/stop"}}})
	if _, still := links.linked[77]; still || !strings.Contains(f.messages()[2], "отключены") {
		t.Fatalf("отвязка не сработала: %v %v", links.linked, f.messages())
	}
}

// Обновления приходят long polling'ом с настоящим разбором JSON клиента MAX.
func TestRunReadsUpdates(t *testing.T) {
	bot, f, links := setup(t)
	f.mu.Lock()
	f.updates = `{"updates":[{"update_type":"bot_started","timestamp":1,"chat_id":5,"user":{"user_id":99,"first_name":"Тест"},"payload":"good"}],"marker":2}`
	f.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go bot.Run(ctx)
	for ctx.Err() == nil {
		links.mu.Lock()
		user := links.linked[99]
		links.mu.Unlock()
		if user == "user-1" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("bot_started из /updates не привязал аккаунт: %v", links.linked)
}
