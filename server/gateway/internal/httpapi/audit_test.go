package httpapi

import (
	"net/http/httptest"
	"testing"

	"example.com/hackathon/apps/gateway/internal/audit"
	"example.com/hackathon/apps/gateway/internal/auth"
	"github.com/gin-gonic/gin"
)

type recorder struct{ got []audit.Record }

func (r *recorder) Record(x audit.Record) { r.got = append(r.got, x) }

func TestAuditTrailRecordsOnlyWrites(t *testing.T) {
	rec := &recorder{}
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "req-1")
		c.Set("identity", auth.Identity{Subject: "u", Email: "u@example.test"})
	}, auditTrail(rec))
	r.GET("/api/v1/tasks", func(c *gin.Context) { c.Status(200) })
	r.POST("/api/v1/tasks/:id", func(c *gin.Context) { c.Status(201) })
	for _, m := range []string{"GET /api/v1/tasks", "POST /api/v1/tasks/42"} {
		method, path, _ := cut(m)
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, path, nil))
	}
	if len(rec.got) != 1 {
		t.Fatalf("записей %d, ожидалась 1 (только изменяющий запрос)", len(rec.got))
	}
	g := rec.got[0]
	if g.ActorID != "u" || g.Route != "/api/v1/tasks/:id" || g.Path != "/api/v1/tasks/42" || g.Status != 201 || g.RequestID != "req-1" {
		t.Fatalf("%+v", g)
	}
}

func cut(s string) (string, string, bool) {
	for i := range s {
		if s[i] == ' ' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}
