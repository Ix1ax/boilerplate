package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"example.com/hackathon/server/gateway/internal/auth"
	"example.com/hackathon/server/gateway/internal/platform"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type oneAdmin struct{ admin auth.Identity }

func (o oneAdmin) AdminLogin(_ context.Context, email, password string) (auth.Identity, error) {
	if email != o.admin.Email || password != "АдминПароль2026" {
		return auth.Identity{}, &auth.UpstreamError{Status: 401, Body: []byte(`{"error":{"code":"invalid_credentials","message":"Неверная почта или пароль"}}`)}
	}
	return o.admin, nil
}
func (o oneAdmin) AdminStatus(context.Context, string) (auth.Identity, error) { return o.admin, nil }
func (o oneAdmin) AdminChangePassword(context.Context, string, string, string) error {
	return nil
}

// Все админские префиксы открываются только админской сессией: Bearer и пользовательская cookie
// получают 401, а сервис видит роль admin только от админа.
func TestAdminRoutesOnlyThroughAdminSession(t *testing.T) {
	const origin = "https://app.example.test"
	var roles []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roles = append(roles, r.Header.Get(HeaderRoles))
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	mr := miniredis.RunT(t)
	cache := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer cache.Close()
	admin := auth.Identity{Subject: "admin-1", Email: "mod@example.test", Name: "Модератор", Provider: auth.ProviderAdmin, PasswordChangedAt: time.Now().UTC()}
	admin.RealmAccess.Roles = []string{"admin"}
	users := auth.NewSessions(cache, nil, "https://id.example.test/realms/r", "", "web", origin, true)
	router := NewGateway(slog.New(slog.NewTextHandler(io.Discard, nil)), &platform.Dependencies{Cache: cache}, nil,
		Upstreams{Tasks: target, Audit: target, Events: target, Moderation: target, Identity: target},
		Options{InternalToken: "secret", AppOrigin: origin, Admin: auth.NewAdminSessions(cache, oneAdmin{admin}, origin, true)}, users)
	gateway := httptest.NewServer(router)
	defer gateway.Close()

	do := func(method, path string, headers map[string]string) *http.Response {
		req, _ := http.NewRequest(method, gateway.URL+path, strings.NewReader(`{"email":"mod@example.test","password":"АдминПароль2026"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}
	paths := []string{"/api/v1/admin/me", "/api/v1/admin/ping", "/api/v1/admin/tasks/stats", "/api/v1/admin/audit",
		"/api/v1/admin/events", "/api/v1/admin/organizations", "/api/v1/admin/comments/x/hide", "/api/v1/admin/sports",
		"/api/v1/admin/domain/republish", "/api/v1/admin/reports", "/api/v1/admin/profiles/x", "/api/v1/admin/moderation/dashboard",
		"/api/v1/admin/accounts", "/api/v1/admin/admins", "/api/v1/admin/reviews/x/hide", "/api/v1/admin/stories/x/media",
		"/api/v1/admin/messages/x/hide"}
	for _, p := range paths {
		for name, h := range map[string]map[string]string{
			"bearer":      {"Authorization": "Bearer token-with-admin-role"},
			"user cookie": {"Cookie": "starter_session=anything"},
		} {
			if resp := do("GET", p, h); resp.StatusCode != 401 {
				t.Fatalf("%s с %s: %d, ждали 401", p, name, resp.StatusCode)
			}
		}
	}
	if len(roles) != 0 {
		t.Fatalf("запросы без админской сессии дошли до сервиса: %v", roles)
	}
	login := do("POST", "/auth/admin/login", nil)
	var cookie string
	for _, c := range login.Cookies() {
		if c.Name == "starter_admin_session" {
			cookie = c.Name + "=" + c.Value
		}
	}
	if login.StatusCode != 200 || cookie == "" {
		t.Fatalf("вход в админку: %d", login.StatusCode)
	}
	for _, p := range paths {
		if resp := do("GET", p, map[string]string{"Cookie": cookie}); resp.StatusCode != 200 {
			t.Fatalf("%s с админской сессией: %d", p, resp.StatusCode)
		}
	}
	for _, got := range roles {
		if got != "admin" {
			t.Fatalf("сервис получил роли %q", got)
		}
	}
	// Подсказка по жалобе вызывает модель: после 10 запросов в минуту — 429, как у /llm.
	for i := 1; i <= 11; i++ {
		resp := do("POST", "/api/v1/admin/reports/x/analyze", map[string]string{"Cookie": cookie})
		if i <= 10 && resp.StatusCode != 200 || i == 11 && resp.StatusCode != 429 {
			t.Fatalf("analyze №%d: %d", i, resp.StatusCode)
		}
	}
}
