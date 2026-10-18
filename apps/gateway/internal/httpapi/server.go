package httpapi

import (
	"context"
	"example.com/hackathon/apps/gateway/internal/apidocs"
	"example.com/hackathon/apps/gateway/internal/audit"
	"example.com/hackathon/apps/gateway/internal/auth"
	"example.com/hackathon/apps/gateway/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Тело запроса ограничено размером крупнейшей загрузки (10 МиБ файла) плюс служебный запас.
const maxBody = 11 << 20

// New собирает gateway без внутренних сервисов: вход, /me и служебные маршруты.
func New(log *slog.Logger, d *platform.Dependencies, verifier auth.Verifier, sessions ...*auth.Sessions) *gin.Engine {
	return NewGateway(log, d, verifier, Upstreams{}, Options{}, sessions...)
}

// NewGateway — единственная публичная точка входа: проверяет сессию или JWT, ограничивает
// частоту изменений и проксирует домены во внутренние сервисы.
// Options — настройки gateway, не связанные с адресами сервисов.
type Options struct {
	InternalToken string
	// Metrics включает GET /metrics (Prometheus). Непустой MetricsToken требует Bearer-токен.
	Metrics      bool
	MetricsToken string
	// Audit — куда отправлять записи журнала (nil — журнал выключен).
	Audit audit.Recorder
}

func NewGateway(log *slog.Logger, d *platform.Dependencies, verifier auth.Verifier, up Upstreams, opts Options, sessions ...*auth.Sessions) *gin.Engine {
	internalToken := opts.InternalToken
	gin.SetMode(gin.ReleaseMode)
	gin.EnableJsonDecoderDisallowUnknownFields()
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(func(c *gin.Context) {
		start := time.Now()
		id := uuid.NewString()
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "no-store")
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)
		c.Next()
		log.Info("HTTP-запрос", "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds(), "request_id", id)
		for _, e := range c.Errors {
			log.Error("Ошибка HTTP-запроса", "request_id", id, "error", e.Err)
		}
	}, gin.CustomRecovery(func(c *gin.Context, _ any) {
		c.AbortWithStatusJSON(500, gin.H{"error": gin.H{"code": "internal_error", "message": "Внутренняя ошибка сервера", "request_id": c.GetString("request_id")}})
	}))
	if opts.Metrics {
		registerMetrics(r, opts.MetricsToken)
	}
	apidocs.Register(r)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		if d == nil || d.Check(ctx) != nil || !upstreamsReady(ctx, up) {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})
	if len(sessions) > 0 && sessions[0] != nil {
		r.Use(sessions[0].CSRF())
		sessions[0].Register(r)
	}
	api := r.Group("/api/v1", auth.Middleware(verifier, sessions...))
	api.GET("/me", func(c *gin.Context) {
		i := auth.Current(c)
		roles := i.RealmAccess.Roles
		if roles == nil {
			roles = []string{}
		}
		c.JSON(200, gin.H{"id": i.Subject, "email": i.Email, "name": i.Name, "roles": roles})
	})
	api.GET("/admin/ping", auth.RequireRole("admin"), func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	if d != nil {
		// Счётчик общий для экземпляров API. Без Redis изменять данные временно нельзя.
		api.Use(limit(d, "rate", 60, true), idempotency(d))
		if opts.Audit != nil {
			api.Use(auditTrail(opts.Audit))
		}
		if up.Tasks != nil {
			h := proxy(up.Tasks, internalToken)
			// ИИ-проверка задачи вызывает модель: на неё действует тот же строгий лимит, что и на /llm.
			costly := limit(d, "rate:llm", 10, false)
			g := api.Group("/tasks", func(c *gin.Context) {
				if c.Request.Method == http.MethodPost && strings.HasSuffix(c.Request.URL.Path, "/analyze") {
					costly(c)
					return
				}
				c.Next()
			})
			g.Any("", h)
			g.Any("/*path", h)
			// Админские отчёты сервиса задач: роль проверяется и здесь, и в самом сервисе.
			api.GET("/admin/tasks/*path", auth.RequireRole("admin"), h)
		}
		if up.LLM != nil {
			// Генерация дорогая (CPU): отдельный, более строгий лимит на любой метод.
			h := proxy(up.LLM, internalToken, slowUpstream)
			g := api.Group("/llm", limit(d, "rate:llm", 10, false))
			g.Any("", h)
			g.Any("/*path", h)
		}
		if up.Audit != nil {
			api.GET("/admin/audit", auth.RequireRole("admin"), proxy(up.Audit, internalToken))
		}
		if up.Notifications != nil {
			// slowUpstream: поток /notifications/stream (SSE) живёт до нескольких минут.
			h := proxy(up.Notifications, internalToken, slowUpstream)
			api.Any("/notifications", h)
			api.Any("/notifications/*path", h)
		}
		if up.Files != nil {
			h := proxy(up.Files, internalToken)
			api.Any("/files", h)
			api.Any("/files/*path", h)
		}
	}
	r.NoRoute(func(c *gin.Context) {
		code := "not_found"
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(404, gin.H{"error": gin.H{"code": code, "message": "Маршрут не найден"}})
			return
		}
		c.Status(404)
	})
	return r
}

func upstreamsReady(ctx context.Context, up Upstreams) bool {
	for _, u := range []*url.URL{up.Tasks, up.Files, up.LLM, up.Notifications, up.Audit} {
		if u == nil {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.ResolveReference(&url.URL{Path: "/readyz"}).String(), nil)
		if err != nil {
			return false
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false
		}
	}
	return true
}

// limit — фиксированное окно в минуту на пользователя. onlyWrites=true пропускает чтение.
func limit(d *platform.Dependencies, prefix string, max int64, onlyWrites bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if onlyWrites && c.Request.Method == http.MethodGet {
			c.Next()
			return
		}
		key := prefix + ":" + auth.Current(c).Subject + ":" + time.Now().UTC().Format("200601021504")
		count, err := d.Cache.Eval(c.Request.Context(), `local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('EXPIRE',KEYS[1],120) end; return n`, []string{key}).Int64()
		if err != nil {
			_ = c.Error(err)
			c.AbortWithStatusJSON(503, gin.H{"error": gin.H{"code": "cache_unavailable", "message": "Попробуйте ещё раз позже"}})
			return
		}
		if count > max {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(429, gin.H{"error": gin.H{"code": "rate_limit", "message": "Слишком много запросов. Подождите минуту"}})
			return
		}
		c.Next()
	}
}
