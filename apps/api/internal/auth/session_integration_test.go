//go:build integration

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

type acceptedToken struct{}

func (acceptedToken) Verify(context.Context, string) (*oidc.IDToken, error) {
	return &oidc.IDToken{}, nil
}

func TestBrowserSessionLifecycle(t *testing.T) {
	cache := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR"), Password: os.Getenv("REDIS_PASSWORD")})
	defer cache.Close()
	if err := cache.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	var exchanges atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") == "authorization_code" && r.Form.Get("code_verifier") == "" {
			t.Error("нет PKCE verifier")
		}
		exchanges.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","expires_in":300,"token_type":"Bearer"}`))
	}))
	defer provider.Close()
	s := NewSessions(cache, acceptedToken{}, provider.URL, "", "web", "https://app.example.test", true)
	router := gin.New()
	s.Register(router)
	request := func(method, path string, cookies []*http.Cookie, origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	started := request("GET", "/auth/login?next=%2Fdashboard%2Ffiles", nil, "")
	if started.Code != 302 {
		t.Fatal(started.Code)
	}
	target, _ := url.Parse(started.Header().Get("Location"))
	if target.Query().Get("code_challenge_method") != "S256" || target.Query().Get("code_challenge") == "" {
		t.Fatal("нет PKCE S256")
	}
	cookies := started.Result().Cookies()
	state := target.Query().Get("state")
	defer cache.Del(context.Background(), "oidc:"+state)
	callback := "/auth/callback?code=test&state=" + state
	if got := request("GET", callback, nil, ""); got.Code != 400 {
		t.Fatal("вход без cookie принят")
	}
	completed := request("GET", callback, cookies, "")
	if completed.Code != 302 || completed.Header().Get("Location") != "/dashboard/files" {
		t.Fatalf("callback: %d %s", completed.Code, completed.Body)
	}
	if got := request("GET", callback, cookies, ""); got.Code != 400 {
		t.Fatal("state принят повторно")
	}
	var sessionCookieValue *http.Cookie
	for _, cookie := range completed.Result().Cookies() {
		if cookie.Name == sessionCookie {
			sessionCookieValue = cookie
		}
	}
	if sessionCookieValue == nil || !sessionCookieValue.HttpOnly || !sessionCookieValue.Secure || strings.Contains(sessionCookieValue.Value, "access") {
		t.Fatal("недопустимая cookie сессии")
	}
	id := sessionCookieValue.Value
	defer cache.Del(context.Background(), "session:"+id)
	// Истёкший access token обновляется один раз для параллельных запросов.
	data, _ := json.Marshal(session{Token: &oauth2.Token{AccessToken: "expired", RefreshToken: "refresh", Expiry: time.Now().Add(-time.Minute)}, Deadline: time.Now().Add(time.Hour)})
	if err := cache.Set(context.Background(), "session:"+id, data, time.Hour).Err(); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 10 {
		group.Go(func() {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/api/v1/me", nil)
			c.Request.AddCookie(sessionCookieValue)
			token, err := s.AccessToken(c)
			if err != nil || token != "access" {
				t.Errorf("refresh: %q %v", token, err)
			}
		})
	}
	group.Wait()
	if exchanges.Load() != 2 {
		t.Fatalf("обновлений: %d", exchanges.Load()-1)
	}
	if got := request("POST", "/auth/refresh", []*http.Cookie{sessionCookieValue}, "https://other.test"); got.Code != 403 {
		t.Fatal("CSRF обновления сессии принят")
	}
	if got := request("POST", "/auth/refresh", nil, s.origin); got.Code != 401 {
		t.Fatal("refresh без cookie принят")
	}
	refreshed := request("POST", "/auth/refresh", []*http.Cookie{sessionCookieValue}, s.origin)
	if refreshed.Code != 200 {
		t.Fatalf("refresh: %d %s", refreshed.Code, refreshed.Body)
	}
	raw, err := cache.Get(context.Background(), "session:"+id).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	var extended session
	if json.Unmarshal(raw, &extended) != nil {
		t.Fatal("не удалось прочитать сессию после refresh")
	}
	if !extended.Deadline.After(time.Now().Add(sessionLifetime - time.Minute)) {
		t.Fatal("refresh не продлил срок сессии")
	}
	if got := request("POST", "/auth/logout", []*http.Cookie{sessionCookieValue}, "https://other.test"); got.Code != 403 {
		t.Fatal("CSRF выхода принят")
	}
	if got := request("POST", "/auth/logout", []*http.Cookie{sessionCookieValue}, s.origin); got.Code != 200 {
		t.Fatal(got.Code)
	}
	if cache.Exists(context.Background(), "session:"+id).Val() != 0 {
		t.Fatal("сессия сохранилась после выхода")
	}
}
