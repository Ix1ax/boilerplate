package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"example.com/hackathon/server/gateway/internal/auth"
	"github.com/gin-gonic/gin"
)

// Сервис должен получить личность только от gateway, а пользовательские заголовки — потерять.
func TestProxyReplacesIdentityHeaders(t *testing.T) {
	var got http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(204)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		id := auth.Identity{Subject: "user-1"}
		id.RealmAccess.Roles = []string{"user"}
		c.Set("identity", id)
	})
	r.Any("/api/v1/tasks", proxy(target, "secret"))
	// ReverseProxy требует настоящий ResponseWriter, поэтому поднимаем сервер, а не Recorder.
	gateway := httptest.NewServer(r)
	defer gateway.Close()
	req, _ := http.NewRequest("GET", gateway.URL+"/api/v1/tasks", nil)
	req.Header.Set(HeaderUserID, "attacker")
	req.Header.Set(HeaderRoles, "admin")
	req.Header.Set(HeaderToken, "guess")
	req.Header.Set("Authorization", "Bearer x")
	req.Header.Set("Cookie", "starter_session=x")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 || got.Get(HeaderUserID) != "user-1" || got.Get(HeaderRoles) != "user" || got.Get(HeaderToken) != "secret" || got.Get("Authorization") != "" || got.Get("Cookie") != "" {
		t.Fatalf("status=%d headers=%v", resp.StatusCode, got)
	}
}

func TestProxyUnavailableUpstream(t *testing.T) {
	target, _ := url.Parse("http://127.0.0.1:1")
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("identity", auth.Identity{Subject: "u"}) })
	r.Any("/x", proxy(target, "t"))
	gateway := httptest.NewServer(r)
	defer gateway.Close()
	resp, err := http.Get(gateway.URL + "/x")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 502 {
		t.Fatalf("ожидался 502, получен %d", resp.StatusCode)
	}
}

// Долгий ответ (SSE) должен доходить частями: проверяем сброс буфера и продление дедлайна записи.
func TestSlowProxyStreamsChunks(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: 1\n\n"))
		w.(http.Flusher).Flush()
		<-release
		_, _ = w.Write([]byte("data: 2\n\n"))
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("identity", auth.Identity{Subject: "u"}) })
	r.Any("/s", proxy(target, "t", slowUpstream))
	gateway := httptest.NewServer(r)
	defer gateway.Close()
	resp, err := http.Get(gateway.URL + "/s")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 9)
	if _, err := resp.Body.Read(buf); err != nil || string(buf) != "data: 1\n\n" {
		t.Fatalf("первый фрагмент не дошёл до разрыва ответа: %q %v", buf, err)
	}
	close(release)
}
