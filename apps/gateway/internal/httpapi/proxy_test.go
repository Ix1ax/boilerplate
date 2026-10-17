package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"example.com/hackathon/apps/gateway/internal/auth"
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
	r.Use(func(c *gin.Context) { c.Set("identity", auth.Identity{Subject: "user-1"}) })
	r.Any("/api/v1/tasks", proxy(target, "secret"))
	// ReverseProxy требует настоящий ResponseWriter, поэтому поднимаем сервер, а не Recorder.
	gateway := httptest.NewServer(r)
	defer gateway.Close()
	req, _ := http.NewRequest("GET", gateway.URL+"/api/v1/tasks", nil)
	req.Header.Set(HeaderUserID, "attacker")
	req.Header.Set(HeaderToken, "guess")
	req.Header.Set("Authorization", "Bearer x")
	req.Header.Set("Cookie", "starter_session=x")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 || got.Get(HeaderUserID) != "user-1" || got.Get(HeaderToken) != "secret" || got.Get("Authorization") != "" || got.Get("Cookie") != "" {
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
