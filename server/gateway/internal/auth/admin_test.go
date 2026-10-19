package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// fakeAdmins — администраторы identity в памяти.
type fakeAdmins struct {
	admins   map[string]Identity
	passes   map[string]string
	inactive map[string]bool
	statuses int
}

func newFakeAdmins() *fakeAdmins {
	f := &fakeAdmins{admins: map[string]Identity{}, passes: map[string]string{}, inactive: map[string]bool{}}
	for email, roles := range map[string][]string{"boss@example.test": {"admin", "superadmin"}, "mod@example.test": {"admin"}} {
		i := Identity{Subject: "id-" + email, Email: email, Name: "Админ", Provider: ProviderAdmin, PasswordChangedAt: time.Now().UTC()}
		i.RealmAccess.Roles = roles
		f.admins[email], f.passes[email] = i, "АдминПароль2026"
	}
	return f
}

func (f *fakeAdmins) AdminLogin(_ context.Context, email, password string) (Identity, error) {
	if f.passes[email] != password || password == "" || f.inactive[email] {
		return Identity{}, &UpstreamError{Status: 401, Body: []byte(`{"error":{"code":"invalid_credentials","message":"Неверная почта или пароль"}}`)}
	}
	return f.admins[email], nil
}

func (f *fakeAdmins) AdminStatus(_ context.Context, id string) (Identity, error) {
	f.statuses++
	for email, i := range f.admins {
		if i.Subject == id && !f.inactive[email] {
			return i, nil
		}
	}
	return Identity{}, ErrAccountInactive
}

func (f *fakeAdmins) AdminChangePassword(_ context.Context, id, current, next string) error {
	for email, i := range f.admins {
		if i.Subject == id {
			if f.passes[email] != current {
				return &UpstreamError{Status: 422, Body: []byte(`{"error":{"code":"validation_error","field":"current_password","message":"Текущий пароль указан неверно"}}`)}
			}
			f.passes[email] = next
			i.PasswordChangedAt = time.Now().UTC()
			f.admins[email] = i
			return nil
		}
	}
	return &UpstreamError{Status: 404, Body: []byte(`{"error":{"code":"not_found","message":"нет"}}`)}
}

func adminRouter(t *testing.T) (*gin.Engine, *fakeAdmins, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	cache := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = cache.Close() })
	admins := newFakeAdmins()
	users := NewSessions(cache, nil, "https://id.example.test/realms/r", "", "web", testOrigin, true)
	s := NewAdminSessions(cache, admins, testOrigin, true)
	r := gin.New()
	r.Use(users.CSRF())
	s.Register(r)
	r.GET("/api/v1/admin/me", s.Middleware(), Me)
	r.POST("/api/v1/admin/action", s.Middleware(), RequireRole("admin"), func(c *gin.Context) { c.Status(204) })
	return r, admins, mr
}

func adminCookieOf(t *testing.T, w interface{ Result() *http.Response }) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == adminCookie && c.Value != "" {
			if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode {
				t.Fatalf("админская cookie без защитных флагов: %+v", c)
			}
			return c
		}
	}
	t.Fatal("нет админской cookie")
	return nil
}

func TestAdminLoginSessionAndLogout(t *testing.T) {
	r, _, _ := adminRouter(t)
	if w := call(r, "POST", "/auth/admin/login", `{"email":"mod@example.test","password":"неверно"}`); w.Code != 401 {
		t.Fatalf("неверный пароль: %d", w.Code)
	}
	if w := call(r, "POST", "/auth/admin/login", `{"email":"mod@example.test","password":"x","role":"superadmin"}`); w.Code != 400 {
		t.Fatalf("лишнее поле role должно отклоняться: %d", w.Code)
	}
	w := call(r, "POST", "/auth/admin/login", `{"email":"mod@example.test","password":"АдминПароль2026"}`)
	if w.Code != 200 {
		t.Fatalf("вход: %d %s", w.Code, w.Body.String())
	}
	cookie := adminCookieOf(t, w)
	me := call(r, "GET", "/api/v1/admin/me", "", cookie)
	var body struct {
		ID    string   `json:"id"`
		Roles []string `json:"roles"`
	}
	if me.Code != 200 || json.Unmarshal(me.Body.Bytes(), &body) != nil || body.ID != "id-mod@example.test" || len(body.Roles) != 1 || body.Roles[0] != "admin" {
		t.Fatalf("me: %d %s", me.Code, me.Body.String())
	}
	if w := call(r, "GET", "/api/v1/admin/me", ""); w.Code != 401 {
		t.Fatalf("без cookie: %d", w.Code)
	}
	// Пользовательская cookie с тем же значением в админку не пускает: другое имя и другое хранилище.
	if w := call(r, "GET", "/api/v1/admin/me", "", &http.Cookie{Name: sessionCookie, Value: cookie.Value}); w.Code != 401 {
		t.Fatalf("пользовательская cookie открыла админку: %d", w.Code)
	}
	if w := call(r, "POST", "/auth/admin/logout", "", cookie); w.Code != 204 {
		t.Fatalf("выход: %d", w.Code)
	}
	if w := call(r, "GET", "/api/v1/admin/me", "", cookie); w.Code != 401 {
		t.Fatalf("после выхода: %d", w.Code)
	}
}

func TestAdminMiddlewareRequiresOriginForWrites(t *testing.T) {
	r, _, _ := adminRouter(t)
	cookie := adminCookieOf(t, call(r, "POST", "/auth/admin/login", `{"email":"mod@example.test","password":"АдминПароль2026"}`))
	if w := call(r, "POST", "/api/v1/admin/action", "", cookie); w.Code != 204 {
		t.Fatalf("действие со своим Origin: %d", w.Code)
	}
	// Authorization отключает общий CSRF-фильтр, но не админский: Origin нужен всегда.
	req := newRequest("POST", "/api/v1/admin/action", "", cookie)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Authorization", "Bearer x")
	if w := serve(r, req); w.Code != 403 {
		t.Fatalf("чужой Origin: %d", w.Code)
	}
}

func TestAdminBlockedAndPasswordChangeCloseSessions(t *testing.T) {
	r, admins, mr := adminRouter(t)
	login := func() *http.Cookie {
		return adminCookieOf(t, call(r, "POST", "/auth/admin/login", `{"email":"mod@example.test","password":"`+admins.passes["mod@example.test"]+`"}`))
	}
	first, second := login(), login()
	if w := call(r, "POST", "/auth/admin/password", `{"current_password":"АдминПароль2026","new_password":"НовыйПароль2027"}`, second); w.Code != 204 {
		t.Fatalf("смена пароля: %d %s", w.Code, w.Body.String())
	}
	if w := call(r, "GET", "/api/v1/admin/me", "", first); w.Code != 401 {
		t.Fatalf("другая сессия осталась открытой: %d", w.Code)
	}
	if w := call(r, "GET", "/api/v1/admin/me", "", second); w.Code != 200 {
		t.Fatalf("текущая сессия закрылась: %d", w.Code)
	}
	// Блокировка вступает в силу при следующей сверке статуса — не дольше adminStatusTTL.
	admins.inactive["mod@example.test"] = true
	mr.FastForward(adminStatusTTL + time.Second)
	if w := call(r, "GET", "/api/v1/admin/me", "", second); w.Code != 401 {
		t.Fatalf("заблокированный админ остался в админке: %d", w.Code)
	}
}

func newRequest(method, path, body string, cookies ...*http.Cookie) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

func serve(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUserRolesNeverGrantAdmin(t *testing.T) {
	got := userRoles([]string{"user", "admin", "superadmin", "offline_access"})
	if len(got) != 2 || got[0] != "user" || got[1] != "offline_access" {
		t.Fatalf("роли пользователя: %v", got)
	}
}
