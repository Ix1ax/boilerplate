package platform

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Контракт с gateway: имена заголовков повторены в каждом сервисе, общего пакета нет.
const (
	HeaderToken  = "X-Internal-Token"
	HeaderUserID = "X-User-ID"
)

// OpenDB открывает пул к базе этого сервиса: чужих таблиц у него нет.
func OpenDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	return pgxpool.NewWithConfig(ctx, cfg)
}

// Owner возвращает владельца записей, которого gateway передал после проверки пользователя.
func Owner(c *gin.Context) string { return c.GetString("owner_id") }

// TrustGateway принимает личность только от gateway: секрет сравнивается за постоянное время,
// владелец берётся из X-User-ID. Запрос в обход gateway получает 401.
func TrustGateway(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.GetHeader(HeaderUserID)
		if token == "" || subtle.ConstantTimeCompare([]byte(c.GetHeader(HeaderToken)), []byte(token)) != 1 || user == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": gin.H{"code": "unauthorized", "message": "Доступ только через gateway"}})
			return
		}
		c.Set("owner_id", user)
		c.Set("roles", c.GetHeader(HeaderRoles))
		c.Request = c.Request.WithContext(WithEmail(c.Request.Context(), c.GetHeader(HeaderEmail)))
		c.Next()
	}
}

// NewEngine создаёт роутер с request-id, логом, healthz и readyz.
func NewEngine(log *slog.Logger, maxBody int64, check func(context.Context) error) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	gin.EnableJsonDecoderDisallowUnknownFields()
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(func(c *gin.Context) {
		start := time.Now()
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = "internal"
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)
		c.Next()
		log.Info("HTTP-запрос", "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds(), "request_id", id)
		for _, e := range c.Errors {
			log.Error("Ошибка HTTP-запроса", "request_id", id, "error", e.Err)
		}
	}, gin.CustomRecovery(func(c *gin.Context, _ any) {
		c.AbortWithStatusJSON(500, gin.H{"error": gin.H{"code": "internal_error", "message": "Внутренняя ошибка сервера", "request_id": c.GetString("request_id")}})
	}))
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		if check == nil || check(ctx) != nil {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})
	return r
}

// HeaderRoles — роли пользователя, которые gateway взял из проверенного JWT.
const HeaderRoles = "X-User-Roles"

// HasRole сообщает, есть ли у пользователя роль. Доверять можно, потому что запрос прошёл TrustGateway.
func HasRole(c *gin.Context, role string) bool {
	for _, r := range strings.Split(c.GetString("roles"), ",") {
		if r == role {
			return true
		}
	}
	return false
}

// RequireRole пропускает только пользователей с ролью.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasRole(c, role) {
			c.AbortWithStatusJSON(403, gin.H{"error": gin.H{"code": "forbidden", "message": "Недостаточно прав для этого действия"}})
			return
		}
		c.Next()
	}
}

// HeaderEmail — адрес пользователя из JWT (может быть пустым).
const HeaderEmail = "X-User-Email"

type emailKey struct{}

// WithEmail кладёт адрес пользователя в контекст: так он доходит до outbox и фоновых обработчиков.
func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, emailKey{}, email)
}

// Email — адрес пользователя из контекста запроса или события.
func Email(ctx context.Context) string {
	v, _ := ctx.Value(emailKey{}).(string)
	return v
}
