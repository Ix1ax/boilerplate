package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"example.com/hackathon/apps/gateway/internal/auth"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// Эхо-сервис вместо realtime: проверяет, что gateway передал личность и не оборвал соединение.
func echoUpstream(t *testing.T) *url.URL {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(HeaderUserID) != "u" || r.Header.Get(HeaderToken) != "t" {
			http.Error(w, "no identity", 401)
			return
		}
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer c.CloseNow()
		for {
			typ, data, err := c.Read(context.Background())
			if err != nil {
				return
			}
			_ = c.Write(context.Background(), typ, data)
		}
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return u
}

func gatewayWS(t *testing.T, timeout time.Duration) string {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/api/v1/ws", func(c *gin.Context) { c.Set("identity", auth.Identity{Subject: "u"}) },
		websocketGuard("https://app.example.test"), proxy(echoUpstream(t), "t"))
	// Таймауты как у боевого gateway, только короче, чтобы тест шёл быстро.
	srv := httptest.NewUnstartedServer(r)
	srv.Config.ReadTimeout, srv.Config.WriteTimeout = timeout, timeout
	srv.Start()
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws"
}

func TestWebsocketSurvivesServerTimeouts(t *testing.T) {
	u := gatewayWS(t, 500*time.Millisecond)
	h := http.Header{}
	h.Set("Origin", "https://app.example.test")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, u, &websocket.DialOptions{HTTPHeader: h})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	time.Sleep(1200 * time.Millisecond) // дольше Read/WriteTimeout сервера
	if err = c.Write(ctx, websocket.MessageText, []byte("ping")); err != nil {
		t.Fatal(err)
	}
	if _, data, err := c.Read(ctx); err != nil || string(data) != "ping" {
		t.Fatalf("соединение оборвано таймаутом сервера: %q %v", data, err)
	}
}

func TestWebsocketRejectsForeignOrigin(t *testing.T) {
	u := gatewayWS(t, time.Second)
	for origin, want := range map[string]int{"https://evil.example": 403, "": 403} {
		h := http.Header{}
		if origin != "" {
			h.Set("Origin", origin)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, resp, err := websocket.Dial(ctx, u, &websocket.DialOptions{HTTPHeader: h})
		cancel()
		if err == nil || resp == nil || resp.StatusCode != want {
			t.Errorf("origin %q: ожидался %d, получено %v", origin, want, err)
		}
	}
}
