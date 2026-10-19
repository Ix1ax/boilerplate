package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Обычная регистрация по почте и паролю — второй способ входа рядом со «Столото ID».
// Пароли и статус учётных записей хранит сервис identity; gateway выдаёт такую же серверную
// сессию в Redis, как после SSO, только вместо токенов Keycloak в ней лежит сам пользователь.

// Accounts — служебные вызовы сервиса identity.
type Accounts interface {
	Register(ctx context.Context, email, password, name string) (Identity, error)
	Login(ctx context.Context, email, password string) (Identity, error)
	// Status возвращает актуальное состояние: ErrAccountInactive — заблокирована или удалена.
	Status(ctx context.Context, subject string) (Identity, error)
	ChangePassword(ctx context.Context, subject, current, next string) error
}

// ErrAccountInactive — учётная запись заблокирована администратором или удалена: сессию закрываем.
var ErrAccountInactive = errors.New("учётная запись недоступна")

// UpstreamError — ответ identity с ошибкой: отдаём клиенту как есть, там понятные сообщения и поля формы.
type UpstreamError struct {
	Status     int
	Body       []byte
	RetryAfter string
}

func (e *UpstreamError) Error() string { return fmt.Sprintf("identity: %d", e.Status) }

// statusTTL — как долго gateway верит, что учётная запись активна. Блокировка вступает в силу
// для открытых сессий не позже этого срока, а identity не получает запрос на каждый вызов API.
const statusTTL = 30 * time.Second

type account struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Roles  []string `json:"roles"`
}

func (a account) identity() Identity {
	i := Identity{Subject: a.ID, Email: a.Email, Name: a.Name, Provider: ProviderLocal}
	i.RealmAccess.Roles = a.Roles
	return i
}

// HTTPAccounts — клиент identity по внутренней сети с общим INTERNAL_TOKEN.
type HTTPAccounts struct {
	base   *url.URL
	token  string
	client *http.Client
}

func NewHTTPAccounts(base *url.URL, token string) *HTTPAccounts {
	// Хеширование Argon2id занимает десятки миллисекунд, под нагрузкой — очередь: запас по времени.
	return &HTTPAccounts{base: base, token: token, client: &http.Client{Timeout: 10 * time.Second}}
}

func (h *HTTPAccounts) call(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, h.base.ResolveReference(&url.URL{Path: path}).String(), reader)
	if err != nil {
		return err
	}
	req.Header.Set("X-Internal-Token", h.token)
	req.Header.Set("Content-Type", "application/json")
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		req.Header.Set("X-Request-ID", id)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return &UpstreamError{Status: resp.StatusCode, Body: raw, RetryAfter: resp.Header.Get("Retry-After")}
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

func (h *HTTPAccounts) Register(ctx context.Context, email, password, name string) (Identity, error) {
	var a account
	err := h.call(ctx, http.MethodPost, "/internal/v1/accounts", map[string]string{"email": email, "password": password, "name": name}, &a)
	return a.identity(), err
}

func (h *HTTPAccounts) Login(ctx context.Context, email, password string) (Identity, error) {
	var a account
	err := h.call(ctx, http.MethodPost, "/internal/v1/login", map[string]string{"email": email, "password": password}, &a)
	return a.identity(), err
}

func (h *HTTPAccounts) Status(ctx context.Context, subject string) (Identity, error) {
	var a account
	err := h.call(ctx, http.MethodGet, "/internal/v1/accounts/"+url.PathEscape(subject), nil, &a)
	var up *UpstreamError
	if errors.As(err, &up) && (up.Status == 404 || up.Status == 400) {
		return Identity{}, ErrAccountInactive
	}
	if err != nil {
		return Identity{}, err
	}
	if a.Status != "active" {
		return Identity{}, ErrAccountInactive
	}
	return a.identity(), nil
}

func (h *HTTPAccounts) ChangePassword(ctx context.Context, subject, current, next string) error {
	return h.call(ctx, http.MethodPost, "/internal/v1/accounts/"+url.PathEscape(subject)+"/password",
		map[string]string{"current_password": current, "new_password": next}, nil)
}

type requestIDKey struct{}

// EnableLocal включает обычную регистрацию и вход по паролю.
func (s *Sessions) EnableLocal(accounts Accounts) { s.accounts = accounts }

// checkLocal сверяет статус учётной записи с identity не чаще раза в statusTTL. Заблокированную
// сессию удаляем сразу: следующий запрос получит 401, а не продолжит работать до конца срока.
func (s *Sessions) checkLocal(ctx context.Context, id string, local *Identity) error {
	key := "account_active:" + local.Subject
	if n, err := s.cache.Exists(ctx, key).Result(); err != nil {
		return err
	} else if n == 1 {
		return nil
	}
	if s.accounts == nil {
		return redis.Nil
	}
	fresh, err := s.accounts.Status(ctx, local.Subject)
	if errors.Is(err, ErrAccountInactive) {
		_ = s.cache.Del(ctx, "session:"+id).Err()
		_ = s.cache.SRem(ctx, sessionsKey(local.Subject), id).Err()
		return redis.Nil
	}
	if err != nil {
		return err
	}
	// Имя и роли берём свежими: так сервисы видят изменения без повторного входа.
	local.Name, local.Email, local.RealmAccess.Roles = fresh.Name, fresh.Email, fresh.RealmAccess.Roles
	return s.cache.Set(ctx, key, "1", statusTTL).Err()
}

// Лимиты защищают identity от перебора и потока регистраций. IP клиента за прокси недоступен,
// поэтому считаем по почте (подбор пароля к одной учётной записи) и общим счётчиком маршрута.
const (
	loginPerEmail   = 10
	loginTotal      = 600
	registerTotal   = 120
	passwordPerUser = 10
)

func (s *Sessions) allow(ctx context.Context, key string, max int64) (bool, error) {
	window := "rate:auth:" + key + ":" + time.Now().UTC().Format("200601021504")
	n, err := s.cache.Eval(ctx, `local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('EXPIRE',KEYS[1],120) end; return n`, []string{window}).Int64()
	return n <= max, err
}

func emailKey(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:12])
}

// readJSON — тело формы входа небольшое: лимит меньше общего, чтобы не гонять мегабайты в Argon2.
func readJSON(c *gin.Context, dst any) bool {
	// Лишние поля (например role) отклоняем явно, не полагаясь на глобальную настройку gin.
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(dst) != nil {
		sessionError(c, 400, "invalid_body", "Проверьте формат данных формы")
		return false
	}
	return true
}

// relay отдаёт клиенту ошибку identity или общую ошибку недоступности.
func relay(c *gin.Context, err error) {
	var up *UpstreamError
	if errors.As(err, &up) && up.Status < 500 && json.Valid(up.Body) {
		if up.RetryAfter != "" {
			c.Header("Retry-After", up.RetryAfter)
		}
		c.Data(up.Status, "application/json; charset=utf-8", up.Body)
		c.Abort()
		return
	}
	_ = c.Error(err)
	sessionError(c, 503, "accounts_unavailable", "Вход по паролю временно недоступен. Попробуйте позже")
}

func (s *Sessions) limited(c *gin.Context, key string, max int64) bool {
	ok, err := s.allow(c.Request.Context(), key, max)
	if err != nil {
		sessionError(c, 503, "session_unavailable", "Вход временно недоступен")
		return true
	}
	if !ok {
		c.Header("Retry-After", "60")
		sessionError(c, 429, "rate_limit", "Слишком много попыток. Подождите минуту")
		return true
	}
	return false
}

func publicUser(i Identity) gin.H {
	roles := i.RealmAccess.Roles
	if roles == nil {
		roles = []string{}
	}
	return gin.H{"id": i.Subject, "email": i.Email, "name": i.Name, "roles": roles, "provider": i.Provider}
}

// startLocal создаёт сессию пользователя обычной регистрации.
func (s *Sessions) startLocal(c *gin.Context, i Identity) bool {
	ctx := c.Request.Context()
	if _, ok := s.save(c, ctx, session{Local: &i, Deadline: time.Now().Add(sessionLifetime)}); !ok {
		return false
	}
	_ = s.cache.Set(ctx, "account_active:"+i.Subject, "1", statusTTL).Err()
	return true
}

// RegisterProviders добавляет GET /auth/providers и, если подключён identity, маршруты /auth/local/*.
// CSRF для POST проверяет общий middleware Sessions.CSRF (точный Origin приложения).
//
//	GET  /auth/providers        какие способы входа доступны — для экрана входа
//	POST /auth/local/register   регистрация и сразу вход
//	POST /auth/local/login      вход по почте и паролю
//	POST /auth/local/password   смена пароля; остальные сессии пользователя закрываются
func (s *Sessions) RegisterProviders(r *gin.Engine) {
	r.GET("/auth/providers", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"sso":   gin.H{"enabled": true, "title": "Столото ID", "login_url": "/auth/login"},
			"local": gin.H{"enabled": s.accounts != nil, "register_url": "/auth/local/register", "login_url": "/auth/local/login"},
		})
	})
	if s.accounts == nil {
		return
	}
	r.POST("/auth/local/register", func(c *gin.Context) {
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Name     string `json:"name"`
		}
		if !readJSON(c, &in) || s.limited(c, "register", registerTotal) {
			return
		}
		i, err := s.accounts.Register(s.ctx(c), in.Email, in.Password, in.Name)
		if err != nil {
			relay(c, err)
			return
		}
		if s.startLocal(c, i) {
			c.JSON(201, gin.H{"user": publicUser(i)})
		}
	})
	r.POST("/auth/local/login", func(c *gin.Context) {
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if !readJSON(c, &in) || s.limited(c, "login", loginTotal) || s.limited(c, "login:"+emailKey(in.Email), loginPerEmail) {
			return
		}
		i, err := s.accounts.Login(s.ctx(c), in.Email, in.Password)
		if err != nil {
			relay(c, err)
			return
		}
		if s.startLocal(c, i) {
			c.JSON(200, gin.H{"user": publicUser(i)})
		}
	})
	r.POST("/auth/local/password", func(c *gin.Context) {
		var in struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if !readJSON(c, &in) {
			return
		}
		local, _, err := s.Authenticate(c)
		if err != nil && !errors.Is(err, redis.Nil) {
			sessionError(c, 503, "session_unavailable", "Сессия временно недоступна. Повторите запрос")
			return
		}
		if local == nil {
			sessionError(c, 401, "unauthorized", "Войдите по почте и паролю, чтобы сменить пароль")
			return
		}
		if s.limited(c, "password:"+local.Subject, passwordPerUser) {
			return
		}
		if err = s.accounts.ChangePassword(s.ctx(c), local.Subject, in.CurrentPassword, in.NewPassword); err != nil {
			relay(c, err)
			return
		}
		// Пароль могли сменить из-за утечки: закрываем все сессии, кроме текущей.
		ctx := c.Request.Context()
		current, _ := c.Cookie(sessionCookie)
		ids, err := s.cache.SMembers(ctx, sessionsKey(local.Subject)).Result()
		if err == nil {
			for _, id := range ids {
				if id != current {
					_ = s.cache.Del(ctx, "session:"+id).Err()
					_ = s.cache.SRem(ctx, sessionsKey(local.Subject), id).Err()
				}
			}
		}
		c.Status(204)
	})
}

// ctx передаёт request_id в identity: цепочку входа видно в логах обоих сервисов.
func (s *Sessions) ctx(c *gin.Context) context.Context {
	return context.WithValue(c.Request.Context(), requestIDKey{}, c.GetString("request_id"))
}
