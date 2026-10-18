package httpapi

import (
	"net/http"
	"time"

	"example.com/hackathon/server/gateway/internal/audit"
	"example.com/hackathon/server/gateway/internal/auth"
	"github.com/gin-gonic/gin"
)

// auditTrail записывает каждый изменяющий запрос аутентифицированного пользователя:
// кто, какой маршрут (шаблон, без значений параметров запроса), с каким результатом.
func auditTrail(rec audit.Recorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			return
		}
		identity := auth.Current(c)
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		rec.Record(audit.Record{
			ActorID: identity.Subject, ActorEmail: identity.Email, Method: c.Request.Method, Route: route,
			Path: c.Request.URL.Path, Status: c.Writer.Status(), RequestID: c.GetString("request_id"),
			IP: c.ClientIP(), OccurredAt: time.Now().UTC(),
		})
	}
}
