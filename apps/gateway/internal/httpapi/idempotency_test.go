package httpapi

import (
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"example.com/hackathon/apps/gateway/internal/auth"
	"example.com/hackathon/apps/gateway/internal/platform"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestIdempotencyReplaysAndGuards(t *testing.T) {
	mr := miniredis.RunT(t)
	d := &platform.Dependencies{Cache: redis.NewClient(&redis.Options{Addr: mr.Addr()})}
	var calls atomic.Int32
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("identity", auth.Identity{Subject: "u"}) }, idempotency(d))
	r.POST("/a", func(c *gin.Context) { calls.Add(1); c.JSON(201, gin.H{"n": calls.Load()}) })
	r.POST("/b", func(c *gin.Context) { c.Status(204) })
	r.POST("/fail", func(c *gin.Context) { calls.Add(1); c.Status(500) })
	do := func(path, key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", path, nil)
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	first, second := do("/a", "k1"), do("/a", "k1")
	if first.Code != 201 || second.Code != 201 || second.Body.String() != first.Body.String() || second.Header().Get("Idempotent-Replayed") != "true" || calls.Load() != 1 {
		t.Fatalf("повтор не воспроизведён: %d %d %q %q calls=%d", first.Code, second.Code, first.Body, second.Body, calls.Load())
	}
	if w := do("/b", "k1"); w.Code != 422 {
		t.Fatalf("ключ для другого запроса: %d", w.Code)
	}
	if do("/a", ""); calls.Load() != 2 {
		t.Fatal("без ключа запрос должен выполняться всегда")
	}
	do("/fail", "k2")
	do("/fail", "k2")
	if calls.Load() != 4 {
		t.Fatal("после ошибки 5xx повтор должен выполняться заново")
	}
}
