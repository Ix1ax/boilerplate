package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Админка — отдельный контур входа: свои учётные записи (identity, таблица admins), своя cookie
// и своя сессия в Redis. Пользовательская сессия и токен «Столото ID» к /api/v1/admin/* не допускаются,
// даже если в токене есть роль admin (Middleware её вырезает).

const (
	adminCookie = "starter_admin_session"
	// adminSessionLifetime — предел с момента входа: рабочий день, потом вход заново.
	adminSessionLifetime = 8 * time.Hour
	// adminStatusTTL короче, чем у пользователей: блокировка и удаление админа должны
	// закрывать доступ к данным всех пользователей почти сразу.
	adminStatusTTL = 10 * time.Second
	// ProviderAdmin — способ входа в Identity для админской сессии.
	ProviderAdmin = "admin"
)

// Лимиты входа в админку: подбор пароля к одной почте и общий поток.
const (
	adminLoginPerEmail   = 10
	adminLoginTotal      = 300
	adminPasswordPerUser = 10
)

// AdminAccounts — служебные вызовы identity для администраторов.
type AdminAccounts interface {
	AdminLogin(ctx context.Context, email, password string) (Identity, error)
	// AdminStatus возвращает актуальное состояние: ErrAccountInactive — заблокирован или удалён.
	AdminStatus(ctx context.Context, id string) (Identity, error)
	AdminChangePassword(ctx context.Context, id, current, next string) error
}

func (h *HTTPAccounts) AdminLogin(ctx context.Context, email, password string) (Identity, error) {
	var a account
	err := h.call(ctx, http.MethodPost, "/internal/v1/admins/login", map[string]string{"email": email, "password": password}, &a)
	return a.admin(), err
}

func (h *HTTPAccounts) AdminStatus(ctx context.Context, id string) (Identity, error) {
	var a account
	err := h.call(ctx, http.MethodGet, "/internal/v1/admins/"+url.PathEscape(id), nil, &a)
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
	return a.admin(), nil
}

func (h *HTTPAccounts) AdminChangePassword(ctx context.Context, id, current, next string) error {
	return h.call(ctx, http.MethodPost, "/internal/v1/admins/"+url.PathEscape(id)+"/password",
		map[string]string{"current_password": current, "new_password": next}, nil)
}

func (a account) admin() Identity {
	i := a.identity()
	i.Provider = ProviderAdmin
	return i
}

type adminSession struct {
	Admin    Identity  `json:"admin"`
	Deadline time.Time `json:"deadline"`
}

// AdminSessions — вход в админку и проверка админской cookie.
type AdminSessions struct {
	cache    *redis.Client
	accounts AdminAccounts
	origin   string
	secure   bool
}

func NewAdminSessions(cache *redis.Client, accounts AdminAccounts, origin string, secure bool) *AdminSessions {
	return &AdminSessions{cache: cache, accounts: accounts, origin: origin, secure: secure}
}

func adminSessionKey(id string) string     { return "admin_session:" + id }
func adminSessionsKey(admin string) string { return "admin_sessions:" + admin }
func adminActiveKey(admin string) string   { return "admin_active:" + admin }

// SameSite=Strict: админке не нужны переходы с чужих сайтов с уже открытой сессией.
func (s *AdminSessions) cookie(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{Name: adminCookie, Value: value, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode})
}

func publicAdmin(i Identity) gin.H {
	roles := i.RealmAccess.Roles
	if roles == nil {
		roles = []string{}
	}
	return gin.H{"id": i.Subject, "email": i.Email, "name": i.Name, "roles": roles}
}

func (s *AdminSessions) limited(c *gin.Context, key string, max int64) bool {
	ok, err := allowRate(c.Request.Context(), s.cache, key, max)
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

// Register добавляет вход, выход и смену пароля админа. CSRF для POST проверяет общий
// middleware Sessions.CSRF; Middleware ниже дополнительно требует Origin для изменений.
//
//	POST /auth/admin/login      вход по почте и паролю администратора
//	POST /auth/admin/logout     выход: сессия удаляется сразу
//	POST /auth/admin/password   смена своего пароля; остальные сессии админа закрываются
func (s *AdminSessions) Register(r *gin.Engine) {
	r.POST("/auth/admin/login", func(c *gin.Context) {
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if !readJSON(c, &in) || s.limited(c, "admin_login", adminLoginTotal) || s.limited(c, "admin_login:"+emailKey(in.Email), adminLoginPerEmail) {
			return
		}
		i, err := s.accounts.AdminLogin(withRequestID(c), in.Email, in.Password)
		if err != nil {
			relay(c, err)
			return
		}
		if s.start(c, i) {
			c.JSON(200, gin.H{"admin": publicAdmin(i)})
		}
	})
	r.POST("/auth/admin/logout", func(c *gin.Context) {
		if c.GetHeader("Origin") != s.origin {
			sessionError(c, 403, "invalid_origin", "Запрос с этого адреса запрещён")
			return
		}
		if id, _ := c.Cookie(adminCookie); id != "" {
			ctx := c.Request.Context()
			if raw, err := s.cache.Get(ctx, adminSessionKey(id)).Bytes(); err == nil {
				var current adminSession
				if json.Unmarshal(raw, &current) == nil {
					_ = s.cache.SRem(ctx, adminSessionsKey(current.Admin.Subject), id).Err()
				}
			}
			if err := s.cache.Del(ctx, adminSessionKey(id)).Err(); err != nil {
				sessionError(c, 503, "session_unavailable", "Не удалось завершить сессию. Повторите выход")
				return
			}
		}
		s.cookie(c, "", -1)
		c.Status(204)
	})
	r.POST("/auth/admin/password", func(c *gin.Context) {
		var in struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if !readJSON(c, &in) {
			return
		}
		id, current, err := s.resolve(c)
		if err != nil && !errors.Is(err, redis.Nil) {
			sessionError(c, 503, "session_unavailable", "Сессия временно недоступна. Повторите запрос")
			return
		}
		if current == nil {
			sessionError(c, 401, "unauthorized", "Войдите в админку, чтобы сменить пароль")
			return
		}
		if s.limited(c, "admin_password:"+current.Subject, adminPasswordPerUser) {
			return
		}
		if err = s.accounts.AdminChangePassword(withRequestID(c), current.Subject, in.CurrentPassword, in.NewPassword); err != nil {
			relay(c, err)
			return
		}
		if err = s.revokeOthers(withRequestID(c), current.Subject, id); err != nil {
			_ = c.Error(err)
			sessionError(c, 503, "session_revocation_unavailable", "Пароль изменён, но завершение других сессий задерживается. Повторите вход через минуту")
			return
		}
		c.Status(204)
	})
}

// start создаёт админскую сессию вместо прежней. Новый идентификатор защищает от фиксации сессии.
func (s *AdminSessions) start(c *gin.Context, i Identity) bool {
	ctx := c.Request.Context()
	id := randomID()
	raw, _ := json.Marshal(adminSession{Admin: i, Deadline: time.Now().Add(adminSessionLifetime)})
	pipe := s.cache.TxPipeline()
	pipe.Set(ctx, adminSessionKey(id), raw, adminSessionLifetime)
	pipe.SAdd(ctx, adminSessionsKey(i.Subject), id)
	pipe.Expire(ctx, adminSessionsKey(i.Subject), adminSessionLifetime)
	pipe.Set(ctx, adminActiveKey(i.Subject), i.PasswordChangedAt.Format(time.RFC3339Nano), adminStatusTTL)
	if old, _ := c.Cookie(adminCookie); old != "" {
		pipe.Del(ctx, adminSessionKey(old))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		sessionError(c, 503, "session_unavailable", "Не удалось сохранить сессию")
		return false
	}
	s.cookie(c, id, int(adminSessionLifetime.Seconds()))
	return true
}

// resolve находит админа по cookie и сверяет его статус с identity не реже раза в adminStatusTTL.
// redis.Nil — сессии нет или она закрыта; другая ошибка — временный сбой.
func (s *AdminSessions) resolve(c *gin.Context) (string, *Identity, error) {
	id, err := c.Cookie(adminCookie)
	if err != nil || id == "" {
		return "", nil, redis.Nil
	}
	ctx := c.Request.Context()
	raw, err := s.cache.Get(ctx, adminSessionKey(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.cookie(c, "", -1)
		}
		return "", nil, err
	}
	var current adminSession
	if json.Unmarshal(raw, &current) != nil || time.Now().After(current.Deadline) {
		s.cookie(c, "", -1)
		return "", nil, s.drop(ctx, id, current.Admin.Subject)
	}
	admin := current.Admin
	cached, err := s.cache.Get(ctx, adminActiveKey(admin.Subject)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", nil, err
	}
	if err == nil {
		if cached != admin.PasswordChangedAt.Format(time.RFC3339Nano) {
			s.cookie(c, "", -1)
			return "", nil, s.drop(ctx, id, admin.Subject)
		}
		return id, &admin, nil
	}
	fresh, err := s.accounts.AdminStatus(withRequestID(c), admin.Subject)
	if errors.Is(err, ErrAccountInactive) || (err == nil && !fresh.PasswordChangedAt.Equal(admin.PasswordChangedAt)) {
		s.cookie(c, "", -1)
		return "", nil, s.drop(ctx, id, admin.Subject)
	}
	if err != nil {
		return "", nil, err
	}
	// Имя и роли берём свежие: суперадминистратор мог их изменить.
	admin.Name, admin.Email, admin.RealmAccess.Roles = fresh.Name, fresh.Email, fresh.RealmAccess.Roles
	if err = s.cache.Set(ctx, adminActiveKey(admin.Subject), admin.PasswordChangedAt.Format(time.RFC3339Nano), adminStatusTTL).Err(); err != nil {
		return "", nil, err
	}
	return id, &admin, nil
}

func (s *AdminSessions) drop(ctx context.Context, id, admin string) error {
	pipe := s.cache.TxPipeline()
	pipe.Del(ctx, adminSessionKey(id))
	if admin != "" {
		pipe.SRem(ctx, adminSessionsKey(admin), id)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	return redis.Nil
}

// revokeOthers закрывает все сессии админа, кроме текущей, и переносит в неё новую версию пароля.
// Обрыв браузерного запроса не должен отменять отзыв: пароль уже сменён в identity.
func (s *AdminSessions) revokeOthers(ctx context.Context, admin, current string) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	fresh, err := s.accounts.AdminStatus(cleanup, admin)
	if err != nil {
		return err
	}
	return s.cache.Eval(cleanup, `
 local raw = redis.call('GET', KEYS[2])
 if raw then
  local data = cjson.decode(raw)
  if data['admin'].sub ~= ARGV[3] then return redis.error_reply('invalid current session') end
  data['admin'].password_changed_at = ARGV[2]
  redis.call('SET', KEYS[2], cjson.encode(data), 'KEEPTTL')
 end
 for _, id in ipairs(redis.call('SMEMBERS', KEYS[1])) do
  if id ~= ARGV[1] then
   redis.call('DEL', 'admin_session:' .. id)
   redis.call('SREM', KEYS[1], id)
  end
 end
 redis.call('SET', KEYS[3], ARGV[2], 'PX', ARGV[4])
 return 1`, []string{adminSessionsKey(admin), adminSessionKey(current), adminActiveKey(admin)}, current, fresh.PasswordChangedAt.Format(time.RFC3339Nano), admin, adminStatusTTL.Milliseconds()).Err()
}

// Middleware пускает в /api/v1/admin/* только по админской cookie. Bearer и пользовательская сессия
// здесь не принимаются. Origin для изменений проверяется всегда — даже если клиент прислал Authorization,
// на который общий CSRF-middleware не смотрит.
func (s *AdminSessions) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead && c.Request.Method != http.MethodOptions && c.GetHeader("Origin") != s.origin {
			sessionError(c, 403, "invalid_origin", "Запрос с этого адреса запрещён")
			return
		}
		_, admin, err := s.resolve(c)
		if err != nil && !errors.Is(err, redis.Nil) {
			sessionError(c, 503, "session_unavailable", "Сессия временно недоступна. Повторите запрос")
			return
		}
		if admin == nil {
			sessionError(c, 401, "unauthorized", "Войдите в админку, чтобы продолжить")
			return
		}
		c.Set("identity", *admin)
		c.Next()
	}
}

// Me — GET /api/v1/admin/me: кто вошёл в админку (маршрут за Middleware).
func Me(c *gin.Context) { c.JSON(200, publicAdmin(Current(c))) }

// withRequestID передаёт request_id в identity: цепочку входа видно в логах обоих сервисов.
func withRequestID(c *gin.Context) context.Context {
	return context.WithValue(c.Request.Context(), requestIDKey{}, c.GetString("request_id"))
}

// allowRate — фиксированное окно в минуту по ключу, общее для всех реплик gateway.
func allowRate(ctx context.Context, cache *redis.Client, key string, max int64) (bool, error) {
	window := "rate:auth:" + key + ":" + time.Now().UTC().Format("200601021504")
	n, err := cache.Eval(ctx, `local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('EXPIRE',KEYS[1],120) end; return n`, []string{window}).Int64()
	return n <= max, err
}
