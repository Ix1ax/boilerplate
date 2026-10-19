package httpapi

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"example.com/hackathon/server/gateway/internal/auth"
	"example.com/hackathon/server/gateway/internal/platform"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestIdempotencyRetriesAfterClientDisconnect(t *testing.T) {
	mr := miniredis.RunT(t)
	d := &platform.Dependencies{Cache: redis.NewClient(&redis.Options{Addr: mr.Addr()})}
	t.Cleanup(func() { _ = d.Cache.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("identity", auth.Identity{Subject: "u"}) }, idempotency(d))
	calls := 0
	r.POST("/a", func(c *gin.Context) {
		calls++
		if calls == 1 {
			cancel()
			c.Status(502)
			return
		}
		c.Status(201)
	})
	for i, expected := range []int{502, 201} {
		req := httptest.NewRequest("POST", "/a", nil)
		if i == 0 {
			req = req.WithContext(ctx)
		}
		req.Header.Set("Idempotency-Key", "disconnected")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != expected {
			t.Fatalf("попытка %d: статус %d, ожидался %d", i, w.Code, expected)
		}
	}
	if calls != 2 {
		t.Fatalf("обработчик вызван %d раз, ожидалось 2", calls)
	}
}

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
