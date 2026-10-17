package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"
)

const sessionCookie = "starter_session"
const stateCookie = "starter_oidc_state"
const sessionLifetime = 8 * time.Hour

type session struct {
	Token    *oauth2.Token `json:"token"`
	Deadline time.Time     `json:"deadline"`
}

type loginState struct {
	Verifier string `json:"verifier"`
	Next     string `json:"next"`
}

// В cookie находится только случайный идентификатор. Токены остаются в Redis;
// удаление сессии при выходе немедленно закрывает доступ через браузер.
type Sessions struct {
	cache    *redis.Client
	verifier Verifier
	oauth    oauth2.Config
	client   *http.Client
	origin   string
	issuer   string
	secure   bool
	refresh  singleflight.Group
}

func NewSessions(cache *redis.Client, verifier Verifier, issuer, internalURL, clientID, origin string, secure bool) *Sessions {
	client := &http.Client{Timeout: 10 * time.Second}
	if internalURL != "" {
		public, _ := url.Parse(issuer)
		internal, _ := url.Parse(internalURL)
		client.Transport = internalTransport{public: public, internal: internal, base: http.DefaultTransport}
	}
	return &Sessions{cache: cache, verifier: verifier, client: client, origin: origin, issuer: issuer, secure: secure,
		oauth: oauth2.Config{ClientID: clientID, RedirectURL: origin + "/auth/callback", Scopes: []string{"openid", "profile", "email"}, Endpoint: oauth2.Endpoint{AuthURL: issuer + "/protocol/openid-connect/auth", TokenURL: issuer + "/protocol/openid-connect/token", AuthStyle: oauth2.AuthStyleInParams}},
	}
}

func randomID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func (s *Sessions) cookie(c *gin.Context, name, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode})
}
func sessionError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
func safeNext(value string) string {
	u, err := url.Parse(value)
	if err != nil || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "\\\r\n") || strings.ContainsAny(u.Path, "\\\r\n") || u.IsAbs() || u.Host != "" || strings.HasPrefix(u.Path, "/auth/") {
		return "/dashboard/tasks"
	}
	return value
}

func (s *Sessions) Register(r *gin.Engine) {
	r.GET("/auth/login", func(c *gin.Context) {
		state := randomID()
		verifier := oauth2.GenerateVerifier()
		data, _ := json.Marshal(loginState{Verifier: verifier, Next: safeNext(c.Query("next"))})
		if err := s.cache.Set(c.Request.Context(), "oidc:"+state, data, 10*time.Minute).Err(); err != nil {
			sessionError(c, 503, "session_unavailable", "Вход временно недоступен")
			return
		}
		s.cookie(c, stateCookie, state, 600)
		c.Redirect(http.StatusFound, s.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)))
	})
	r.GET("/auth/callback", func(c *gin.Context) {
		cookie, _ := c.Cookie(stateCookie)
		s.cookie(c, stateCookie, "", -1)
		if cookie == "" || cookie != c.Query("state") {
			sessionError(c, 400, "invalid_state", "Не удалось проверить вход. Начните заново")
			return
		}
		raw, err := s.cache.GetDel(c.Request.Context(), "oidc:"+cookie).Bytes()
		var state loginState
		if err != nil || json.Unmarshal(raw, &state) != nil || c.Query("code") == "" {
			sessionError(c, 400, "invalid_state", "Срок входа истёк или вход отменён. Начните заново")
			return
		}
		ctx := context.WithValue(c.Request.Context(), oauth2.HTTPClient, s.client)
		token, err := s.oauth.Exchange(ctx, c.Query("code"), oauth2.VerifierOption(state.Verifier))
		if err != nil {
			sessionError(c, 502, "login_failed", "Не удалось завершить вход")
			return
		}
		if _, err = s.verifier.Verify(ctx, token.AccessToken); err != nil {
			sessionError(c, 401, "invalid_token", "Сервер входа выдал недопустимый токен")
			return
		}
		id := randomID()
		data, _ := json.Marshal(session{Token: token, Deadline: time.Now().Add(sessionLifetime)})
		if err = s.cache.Set(ctx, "session:"+id, data, sessionLifetime).Err(); err != nil {
			sessionError(c, 503, "session_unavailable", "Не удалось сохранить сессию")
			return
		}
		if old, _ := c.Cookie(sessionCookie); old != "" {
			_ = s.cache.Del(ctx, "session:"+old).Err()
		}
		s.cookie(c, sessionCookie, id, int(sessionLifetime.Seconds()))
		c.Redirect(http.StatusFound, state.Next)
	})
	r.POST("/auth/refresh", func(c *gin.Context) {
		if c.GetHeader("Origin") != s.origin {
			sessionError(c, 403, "invalid_origin", "Запрос с этого адреса запрещён")
			return
		}
		id, err := c.Cookie(sessionCookie)
		if err != nil || id == "" {
			sessionError(c, 401, "unauthorized", "Войдите, чтобы продолжить")
			return
		}
		ctx := c.Request.Context()
		raw, err := s.cache.Get(ctx, "session:"+id).Bytes()
		var current session
		if err != nil || json.Unmarshal(raw, &current) != nil || current.Token == nil || time.Now().After(current.Deadline) {
			s.cookie(c, sessionCookie, "", -1)
			sessionError(c, 401, "unauthorized", "Войдите, чтобы продолжить")
			return
		}
		// Продлеваем абсолютный срок сессии (sliding window) при активности пользователя.
		current.Deadline = time.Now().Add(sessionLifetime)
		data, _ := json.Marshal(current)
		saved, err := s.cache.SetXX(ctx, "session:"+id, data, sessionLifetime).Result()
		if err != nil {
			sessionError(c, 503, "session_unavailable", "Не удалось обновить сессию")
			return
		}
		if !saved {
			// SET XX не позволяет запоздалому refresh восстановить удалённую при выходе сессию.
			s.cookie(c, sessionCookie, "", -1)
			sessionError(c, 401, "unauthorized", "Войдите, чтобы продолжить")
			return
		}
		s.cookie(c, sessionCookie, id, int(sessionLifetime.Seconds()))
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.POST("/auth/logout", func(c *gin.Context) {
		if c.GetHeader("Origin") != s.origin {
			sessionError(c, 403, "invalid_origin", "Запрос с этого адреса запрещён")
			return
		}
		if id, _ := c.Cookie(sessionCookie); id != "" {
			if err := s.cache.Del(c.Request.Context(), "session:"+id).Err(); err != nil {
				sessionError(c, 503, "session_unavailable", "Не удалось завершить сессию. Повторите выход")
				return
			}
		}
		s.cookie(c, sessionCookie, "", -1)
		params := url.Values{"client_id": {s.oauth.ClientID}, "post_logout_redirect_uri": {s.origin + "/"}}
		c.JSON(200, gin.H{"url": s.issuer + "/protocol/openid-connect/logout?" + params.Encode()})
	})
}

// Bearer-клиенты не используют браузерную сессию. Для cookie-запросов,
// изменяющих данные, требуется точный Origin приложения.
func (s *Sessions) CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS" && c.GetHeader("Authorization") == "" && c.GetHeader("Origin") != s.origin {
			sessionError(c, 403, "invalid_origin", "Запрос с этого адреса запрещён")
			return
		}
		c.Next()
	}
}

func (s *Sessions) AccessToken(c *gin.Context) (string, error) {
	id, err := c.Cookie(sessionCookie)
	if err != nil || id == "" {
		return "", redis.Nil
	}
	result, err, _ := s.refresh.Do(id, func() (any, error) {
		ctx := context.WithValue(c.Request.Context(), oauth2.HTTPClient, s.client)
		raw, err := s.cache.Get(ctx, "session:"+id).Bytes()
		if err != nil {
			return "", err
		}
		var current session
		if json.Unmarshal(raw, &current) != nil || current.Token == nil || time.Now().After(current.Deadline) {
			return "", redis.Nil
		}
		if current.Token.Expiry.After(time.Now().Add(30 * time.Second)) {
			return current.Token.AccessToken, nil
		}
		token, err := s.oauth.TokenSource(ctx, current.Token).Token()
		if err != nil {
			var response *oauth2.RetrieveError
			if errors.As(err, &response) && response.Response.StatusCode == 400 {
				_ = s.cache.Del(ctx, "session:"+id).Err()
				return "", redis.Nil
			}
			return "", err
		}
		current.Token = token
		data, _ := json.Marshal(current)
		// SET XX не позволяет запоздалому refresh восстановить удалённую при выходе сессию.
		saved, err := s.cache.SetXX(ctx, "session:"+id, data, time.Until(current.Deadline)).Result()
		if err != nil {
			return "", err
		}
		if !saved {
			return "", redis.Nil
		}
		return token.AccessToken, nil
	})
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.cookie(c, sessionCookie, "", -1)
		}
		return "", err
	}
	return result.(string), nil
}
