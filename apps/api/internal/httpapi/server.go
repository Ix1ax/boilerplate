package httpapi

import (
	"context"
	"example.com/hackathon/apps/api/internal/auth"
	"example.com/hackathon/apps/api/internal/files"
	"example.com/hackathon/apps/api/internal/platform"
	"example.com/hackathon/apps/api/internal/tasks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func New(log *slog.Logger, d *platform.Dependencies, verifier auth.Verifier) *gin.Engine {
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
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, files.MaxSize+(1<<20))
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
		if d == nil || d.Check(ctx) != nil {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})
	api := r.Group("/api/v1", auth.Middleware(verifier))
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
		api.Use(func(c *gin.Context) {
			if c.Request.Method == http.MethodGet {
				c.Next()
				return
			}
			key := "rate:" + auth.Current(c).Subject + ":" + time.Now().UTC().Format("200601021504")
			count, err := d.Cache.Eval(c.Request.Context(), `local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('EXPIRE',KEYS[1],120) end; return n`, []string{key}).Int64()
			if err != nil {
				_ = c.Error(err)
				c.AbortWithStatusJSON(503, gin.H{"error": gin.H{"code": "cache_unavailable", "message": "Попробуйте ещё раз позже"}})
				return
			}
			if count > 60 {
				c.Header("Retry-After", "60")
				c.AbortWithStatusJSON(429, gin.H{"error": gin.H{"code": "rate_limit", "message": "Слишком много запросов. Подождите минуту"}})
				return
			}
			c.Next()
		})
		tasks.Register(api, tasks.New(tasks.NewRepository(d.DB)))
		files.Register(api, d)
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
