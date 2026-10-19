package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// fakeAccounts — identity в памяти: проверяем поведение gateway, а не хеширование.
type fakeAccounts struct {
	users    map[string]Identity
	passes   map[string]string
	blocked  map[string]bool
	statuses int
}

func newFakeAccounts() *fakeAccounts {
	return &fakeAccounts{users: map[string]Identity{}, passes: map[string]string{}, blocked: map[string]bool{}}
}

func (f *fakeAccounts) Register(_ context.Context, email, password, name string) (Identity, error) {
	if _, ok := f.users[email]; ok {
		return Identity{}, &UpstreamError{Status: 409, Body: []byte(`{"error":{"code":"email_taken","message":"Эта почта уже зарегистрирована"}}`)}
	}
	i := Identity{Subject: "id-" + email, Email: email, Name: name, Provider: ProviderLocal}
	i.RealmAccess.Roles = []string{"user"}
	f.users[email], f.passes[email] = i, password
	return i, nil
}

func (f *fakeAccounts) Login(_ context.Context, email, password string) (Identity, error) {
	if f.passes[email] != password || password == "" {
		return Identity{}, &UpstreamError{Status: 401, Body: []byte(`{"error":{"code":"invalid_credentials","message":"Неверная почта или пароль"}}`)}
	}
	return f.users[email], nil
}

func (f *fakeAccounts) Status(_ context.Context, subject string) (Identity, error) {
	f.statuses++
	for email, i := range f.users {
		if i.Subject == subject {
			if f.blocked[email] {
				return Identity{}, ErrAccountInactive
			}
			return i, nil
		}
	}
	return Identity{}, ErrAccountInactive
}

func (f *fakeAccounts) ChangePassword(_ context.Context, subject, current, next string) error {
	for email, i := range f.users {
		if i.Subject == subject {
			if f.passes[email] != current {
				return &UpstreamError{Status: 422, Body: []byte(`{"error":{"code":"validation_error","field":"current_password","message":"Текущий пароль указан неверно"}}`)}
			}
			f.passes[email] = next
			return nil
		}
	}
	return &UpstreamError{Status: 404, Body: []byte(`{"error":{"code":"not_found","message":"нет"}}`)}
}

const testOrigin = "https://app.example.test"

func localRouter(t *testing.T) (*gin.Engine, *fakeAccounts, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	cache := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = cache.Close() })
	accounts := newFakeAccounts()
	s := NewSessions(cache, nil, "https://id.example.test/realms/r", "", "web", testOrigin, true)
	s.EnableLocal(accounts)
	r := gin.New()
	r.Use(s.CSRF())
	s.Register(r)
	s.RegisterProviders(r)
	r.GET("/api/v1/me", Middleware(nil, s), func(c *gin.Context) { c.JSON(200, Current(c)) })
	return r, accounts, mr
}

func call(r *gin.Engine, method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", testOrigin)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func sessionOf(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie && c.Value != "" {
			if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
				t.Fatalf("cookie без защитных флагов: %+v", c)
			}
			return c
		}
	}
	t.Fatalf("нет cookie сессии: %d %s", w.Code, w.Body.String())
	return nil
}

func TestLocalRegistrationLoginAndLogout(t *testing.T) {
	r, _, _ := localRouter(t)
	w := call(r, "GET", "/auth/providers", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"local":{"enabled":true`) || !strings.Contains(w.Body.String(), "Столото ID") {
		t.Fatalf("providers: %s", w.Body.String())
	}
	w = call(r, "POST", "/auth/local/register", `{"email":"anna@example.ru","password":"Зарядка2026","name":"Анна"}`)
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"provider":"local"`) {
		t.Fatalf("регистрация: %d %s", w.Code, w.Body.String())
	}
	cookie := sessionOf(t, w)
	w = call(r, "GET", "/api/v1/me", "", cookie)
	var me Identity
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &me) != nil || me.Subject != "id-anna@example.ru" || me.Provider != ProviderLocal || me.RealmAccess.Roles[0] != "user" {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}
	// Ошибка identity (поле формы, код) доходит до клиента без изменений.
	if w = call(r, "POST", "/auth/local/register", `{"email":"anna@example.ru","password":"x","name":"А"}`); w.Code != 409 || !strings.Contains(w.Body.String(), "email_taken") {
		t.Fatalf("повторная регистрация: %d %s", w.Code, w.Body.String())
	}
	if w = call(r, "POST", "/auth/local/login", `{"email":"anna@example.ru","password":"не тот"}`); w.Code != 401 {
		t.Fatalf("неверный пароль: %d", w.Code)
	}
	w = call(r, "POST", "/auth/logout", "", cookie)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"url":"`+testOrigin+`/"`) {
		t.Fatalf("выход локальной сессии должен вести в приложение, а не в Keycloak: %s", w.Body.String())
	}
	if w = call(r, "GET", "/api/v1/me", "", cookie); w.Code != 401 {
		t.Fatalf("после выхода сессия жива: %d", w.Code)
	}
	if w = call(r, "POST", "/auth/local/login", `{"email":"anna@example.ru","password":"Зарядка2026"}`); w.Code != 200 {
		t.Fatalf("вход: %d %s", w.Code, w.Body.String())
	}
}

func TestLocalRequiresOriginAndValidBody(t *testing.T) {
	r, _, _ := localRouter(t)
	req := httptest.NewRequest("POST", "/auth/local/login", strings.NewReader(`{"email":"a@b.ru","password":"x"}`))
	req.Header.Set("Origin", "https://evil.test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("чужой Origin: %d", w.Code)
	}
	if w = call(r, "POST", "/auth/local/login", `{"email":"a@b.ru","password":"x","role":"admin"}`); w.Code != 400 {
		t.Fatalf("лишнее поле должно отклоняться: %d", w.Code)
	}
	if w = call(r, "POST", "/auth/local/login", `{"email":"`+strings.Repeat("a", 10000)+`"}`); w.Code != 400 {
		t.Fatalf("большое тело: %d", w.Code)
	}
}

func TestBlockedAccountLosesSession(t *testing.T) {
	r, accounts, mr := localRouter(t)
	cookie := sessionOf(t, call(r, "POST", "/auth/local/register", `{"email":"igor@example.ru","password":"Пробежка2026","name":"Игорь"}`))
	if w := call(r, "GET", "/api/v1/me", "", cookie); w.Code != 200 || accounts.statuses != 0 {
		t.Fatalf("свежая сессия не должна сверяться сразу: %d, сверок %d", w.Code, accounts.statuses)
	}
	accounts.blocked["igor@example.ru"] = true
	mr.FastForward(statusTTL + 1)
	if w := call(r, "GET", "/api/v1/me", "", cookie); w.Code != 401 {
		t.Fatalf("заблокированный пользователь сохранил доступ: %d", w.Code)
	}
	accounts.blocked["igor@example.ru"] = false
	if w := call(r, "GET", "/api/v1/me", "", cookie); w.Code != 401 {
		t.Fatalf("закрытая сессия не должна оживать после разблокировки: %d", w.Code)
	}
}

func TestPasswordChangeRevokesOtherSessions(t *testing.T) {
	r, _, _ := localRouter(t)
	first := sessionOf(t, call(r, "POST", "/auth/local/register", `{"email":"olga@example.ru","password":"Йога2026утро","name":"Ольга"}`))
	second := sessionOf(t, call(r, "POST", "/auth/local/login", `{"email":"olga@example.ru","password":"Йога2026утро"}`))
	if w := call(r, "POST", "/auth/local/password", `{"current_password":"неверный","new_password":"Йога2027вечер"}`, second); w.Code != 422 || !strings.Contains(w.Body.String(), "current_password") {
		t.Fatalf("неверный текущий пароль: %d %s", w.Code, w.Body.String())
	}
	if w := call(r, "POST", "/auth/local/password", `{"current_password":"Йога2026утро","new_password":"Йога2027вечер"}`); w.Code != 401 {
		t.Fatalf("смена пароля без сессии: %d", w.Code)
	}
	if w := call(r, "POST", "/auth/local/password", `{"current_password":"Йога2026утро","new_password":"Йога2027вечер"}`, second); w.Code != 204 {
		t.Fatalf("смена пароля: %d %s", w.Code, w.Body.String())
	}
	if w := call(r, "GET", "/api/v1/me", "", first); w.Code != 401 {
		t.Fatalf("другая сессия не закрыта: %d", w.Code)
	}
	if w := call(r, "GET", "/api/v1/me", "", second); w.Code != 200 {
		t.Fatalf("текущая сессия закрыта: %d", w.Code)
	}
}

func TestLoginRateLimitPerEmail(t *testing.T) {
	r, _, _ := localRouter(t)
	for i := range loginPerEmail {
		if w := call(r, "POST", "/auth/local/login", `{"email":"Target@Example.ru","password":"guess"}`); w.Code != 401 {
			t.Fatalf("попытка %d: %d", i, w.Code)
		}
	}
	// Регистр и пробелы не помогают обойти лимит.
	if w := call(r, "POST", "/auth/local/login", `{"email":" target@example.ru","password":"guess"}`); w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatalf("лимит по почте: %d", w.Code)
	}
	if w := call(r, "POST", "/auth/local/login", `{"email":"other@example.ru","password":"guess"}`); w.Code != 401 {
		t.Fatalf("лимит задел другую почту: %d", w.Code)
	}
}

func TestProvidersWithoutIdentity(t *testing.T) {
	s := NewSessions(redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()}), nil, "https://id.example.test", "", "web", testOrigin, true)
	r := gin.New()
	s.RegisterProviders(r)
	if w := call(r, "GET", "/auth/providers", ""); !strings.Contains(w.Body.String(), `"local":{"enabled":false`) {
		t.Fatalf("providers: %s", w.Body.String())
	}
	if w := call(r, "POST", "/auth/local/login", `{}`); w.Code != 404 {
		t.Fatalf("маршрут без identity: %d", w.Code)
	}
}
