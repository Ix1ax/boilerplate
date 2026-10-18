package realtime

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/hackathon/server/realtime/internal/platform"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

func server(t *testing.T) (string, *Hub) {
	gin.SetMode(gin.ReleaseMode)
	hub := NewHub()
	r := gin.New()
	Register(r.Group("/api/v1", platform.TrustGateway("secret")), hub, LocalPublisher(hub), slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws", hub
}

type frame struct {
	Type    string `json:"type"`
	Payload struct {
		Channel string          `json:"channel"`
		From    string          `json:"from"`
		Data    json.RawMessage `json:"data"`
		Code    string          `json:"code"`
	} `json:"payload"`
}

func dial(t *testing.T, url, user, roles string) *websocket.Conn {
	h := http.Header{}
	h.Set(platform.HeaderToken, "secret")
	h.Set(platform.HeaderUserID, user)
	h.Set(platform.HeaderRoles, roles)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: h})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.CloseNow() })
	return c
}

func send(t *testing.T, c *websocket.Conn, msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageText, []byte(msg)); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, c *websocket.Conn) frame {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var f frame
	_ = json.Unmarshal(data, &f)
	return f
}

// waitSubs ждёт, пока подписка обработается сервером (команды асинхронны).
func waitSubs(t *testing.T, hub *Hub, channel string, n int) {
	for i := 0; i < 100 && hub.Subscribers(channel) < n; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.Subscribers(channel) < n {
		t.Fatalf("подписка на %s не появилась", channel)
	}
}

func TestPingAndRoomChat(t *testing.T) {
	url, hub := server(t)
	a, b := dial(t, url, "alice", "user"), dial(t, url, "bob", "user")
	send(t, a, `{"type":"ping"}`)
	if f := read(t, a); f.Type != "pong" {
		t.Fatalf("%+v", f)
	}
	send(t, a, `{"type":"subscribe","payload":{"channel":"room:demo"}}`)
	waitSubs(t, hub, "room:demo", 1)
	send(t, b, `{"type":"publish","payload":{"channel":"room:demo","data":{"text":"привет"}}}`)
	f := read(t, a)
	if f.Type != "message" || f.Payload.Channel != "room:demo" || f.Payload.From != "bob" || string(f.Payload.Data) != `{"text":"привет"}` {
		t.Fatalf("%+v %s", f, f.Payload.Data)
	}
}

func TestChannelPolicy(t *testing.T) {
	url, hub := server(t)
	u := dial(t, url, "alice", "user")
	for _, tc := range []struct{ msg, code string }{
		{`{"type":"subscribe","payload":{"channel":"user:bob"}}`, "forbidden"},
		{`{"type":"subscribe","payload":{"channel":"secret"}}`, "forbidden"},
		{`{"type":"publish","payload":{"channel":"broadcast","data":{}}}`, "forbidden"},
		{`{"type":"publish","payload":{"channel":"user:alice","data":{}}}`, "forbidden"},
		{`{"type":"publish","payload":{"channel":"room:x"}}`, "invalid_message"},
		{`не json`, "invalid_message"},
		{`{"type":"hack"}`, "unknown_type"},
	} {
		send(t, u, tc.msg)
		if f := read(t, u); f.Type != "error" || f.Payload.Code != tc.code {
			t.Errorf("%s: %+v", tc.msg, f)
		}
	}
	// Личный канал и объявления администратора разрешены.
	send(t, u, `{"type":"subscribe","payload":{"channel":"user:alice"}}`)
	send(t, u, `{"type":"subscribe","payload":{"channel":"broadcast"}}`)
	waitSubs(t, hub, "broadcast", 1)
	admin := dial(t, url, "root", "user,admin")
	send(t, admin, `{"type":"publish","payload":{"channel":"broadcast","data":{"text":"всем"}}}`)
	if f := read(t, u); f.Type != "message" || f.Payload.From != "root" {
		t.Fatalf("%+v", f)
	}
	// Сервер (через Hub) пишет в личный канал.
	hub.Deliver(Message{Channel: "user:alice", From: "system", Data: json.RawMessage(`{"n":1}`)})
	if f := read(t, u); f.Payload.Channel != "user:alice" {
		t.Fatalf("%+v", f)
	}
}

func TestRateLimitAndFrameSize(t *testing.T) {
	url, _ := server(t)
	c := dial(t, url, "alice", "user")
	for i := 0; i < 60; i++ {
		send(t, c, `{"type":"ping"}`)
	}
	limited := false
	for i := 0; i < 60 && !limited; i++ {
		limited = read(t, c).Payload.Code == "rate_limit"
	}
	if !limited {
		t.Fatal("частые сообщения не ограничены")
	}
	big := dial(t, url, "bob", "user")
	send(t, big, `{"type":"publish","payload":{"channel":"room:x","data":"`+strings.Repeat("я", 20000)+`"}}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, _, err := big.Read(ctx); websocket.CloseStatus(err) != websocket.StatusMessageTooBig {
		t.Fatalf("крупное сообщение не отклонено: %v", err)
	}
}

func TestRejectsWithoutGateway(t *testing.T) {
	url, _ := server(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, resp, err := websocket.Dial(ctx, url, nil)
	if err == nil || resp == nil || resp.StatusCode != 401 {
		t.Fatalf("прямое подключение без gateway должно получить 401: %v", err)
	}
}
