//go:build integration

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Быстрый вход должен давать ту же сессию, что и вход через форму: токен по паролю аккаунта,
// cookie с идентификатором и рабочий AccessToken.
func TestDemoLoginCreatesSession(t *testing.T) {
	cache := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR"), Password: os.Getenv("REDIS_PASSWORD")})
	defer cache.Close()
	if err := cache.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "password" || r.Form.Get("username") != "anna@example.test" || r.Form.Get("password") != "secret-pass" {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"demo-access","refresh_token":"refresh","expires_in":300,"token_type":"Bearer"}`))
	}))
	defer provider.Close()
	s := NewSessions(cache, acceptedToken{}, provider.URL, "", "web", "https://app.example.test", true)
	router := gin.New()
	s.RegisterDemo(router, []DemoAccount{
		{Key: "anna", email: "anna@example.test", pass: "secret-pass"},
		{Key: "broken", email: "anna@example.test", pass: "wrong"},
	})
	login := func(key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/auth/demo/"+key, nil)
		req.Header.Set("Origin", "https://app.example.test")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	if got := login("broken"); got.Code != 409 {
		t.Fatalf("неверный пароль демо-аккаунта: %d", got.Code)
	}
	done := login("anna")
	if done.Code != 200 {
		t.Fatalf("%d %s", done.Code, done.Body)
	}
	var cookie *http.Cookie
	for _, c := range done.Result().Cookies() {
		if c.Name == sessionCookie {
			cookie = c
		}
	}
	if cookie == nil || !cookie.HttpOnly {
		t.Fatal("нет cookie сессии")
	}
	defer cache.Del(context.Background(), "session:"+cookie.Value)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/me", nil)
	c.Request.AddCookie(cookie)
	if token, err := s.AccessToken(c); err != nil || token != "demo-access" {
		t.Fatalf("сессия демо-входа не работает: %q %v", token, err)
	}
}
