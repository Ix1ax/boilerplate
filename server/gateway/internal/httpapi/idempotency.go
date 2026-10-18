package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"example.com/hackathon/apps/gateway/internal/auth"
	"example.com/hackathon/apps/gateway/internal/platform"
	"github.com/gin-gonic/gin"
)

const (
	idempotencyTTL     = 24 * time.Hour
	idempotencyMaxBody = 1 << 20
)

type storedResponse struct {
	Fingerprint string `json:"f"`
	Pending     bool   `json:"p,omitempty"`
	Status      int    `json:"s,omitempty"`
	ContentType string `json:"c,omitempty"`
	Body        []byte `json:"b,omitempty"`
}

// teeWriter копирует тело ответа, не мешая потоковой передаче клиенту.
type teeWriter struct {
	gin.ResponseWriter
	buf      bytes.Buffer
	overflow bool
}

func (w *teeWriter) Write(b []byte) (int, error) {
	if !w.overflow {
		if w.buf.Len()+len(b) > idempotencyMaxBody {
			w.overflow = true
		} else {
			w.buf.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}
func (w *teeWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

// idempotency защищает изменяющие запросы от двойной отправки (двойной клик, повтор после
// обрыва сети). Клиент передаёт Idempotency-Key; повтор с тем же ключом получает сохранённый
// ответ с заголовком Idempotent-Replayed. Без заголовка запросы проходят как обычно.
func idempotency(d *platform.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" || c.Request.Method == http.MethodGet || strings.HasSuffix(c.Request.URL.Path, "/stream") {
			c.Next()
			return
		}
		if len(key) > 100 {
			c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"code": "invalid_idempotency_key", "message": "Idempotency-Key длиннее 100 символов"}})
			return
		}
		ctx := c.Request.Context()
		redisKey := "idem:" + auth.Current(c).Subject + ":" + key
		fingerprint := c.Request.Method + " " + c.Request.URL.Path
		pending, _ := json.Marshal(storedResponse{Fingerprint: fingerprint, Pending: true})
		created, err := d.Cache.SetNX(ctx, redisKey, pending, idempotencyTTL).Result()
		if err != nil {
			_ = c.Error(err)
			c.AbortWithStatusJSON(503, gin.H{"error": gin.H{"code": "cache_unavailable", "message": "Попробуйте ещё раз позже"}})
			return
		}
		if !created {
			var prev storedResponse
			raw, _ := d.Cache.Get(ctx, redisKey).Bytes()
			_ = json.Unmarshal(raw, &prev)
			switch {
			case prev.Fingerprint != fingerprint:
				c.AbortWithStatusJSON(422, gin.H{"error": gin.H{"code": "idempotency_key_reused", "message": "Этот Idempotency-Key уже использован для другого запроса"}})
			case prev.Pending:
				c.AbortWithStatusJSON(409, gin.H{"error": gin.H{"code": "request_in_progress", "message": "Такой запрос уже выполняется"}})
			default:
				c.Header("Idempotent-Replayed", "true")
				c.Data(prev.Status, prev.ContentType, prev.Body)
				c.Abort()
			}
			return
		}
		tee := &teeWriter{ResponseWriter: c.Writer}
		c.Writer = tee
		c.Next()
		status := tee.Status()
		// Ошибку сервера можно повторить: ключ освобождаем. Слишком большой ответ не кешируем.
		if status >= 500 || tee.overflow {
			d.Cache.Del(ctx, redisKey)
			return
		}
		done, _ := json.Marshal(storedResponse{Fingerprint: fingerprint, Status: status, ContentType: tee.Header().Get("Content-Type"), Body: tee.buf.Bytes()})
		d.Cache.Set(context.WithoutCancel(ctx), redisKey, done, idempotencyTTL)
	}
}
