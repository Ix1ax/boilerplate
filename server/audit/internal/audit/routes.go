package audit

import (
	"context"
	"strconv"

	"example.com/hackathon/server/audit/internal/platform"
	"github.com/gin-gonic/gin"
)

func itoa(n int) string { return strconv.Itoa(n) }

// Lister — чтение журнала (Repository).
type Lister interface {
	List(ctx context.Context, q platform.ListQuery) (Page, error)
}

var listSpec = platform.ListSpec{
	Filters:     map[string]string{"actor": "actor", "method": "method", "status": "status::text", "request_id": "request_id"},
	Sorts:       map[string]string{"occurred": "occurred_at", "status": "status"},
	DefaultSort: "-occurred",
	Search:      []string{"path", "actor_email"},
}

func Register(group *gin.RouterGroup, l Lister) {
	group.GET("/admin/audit", platform.RequireRole("admin"), func(c *gin.Context) {
		q, err := platform.ParseList(c, listSpec, 1)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"code": "invalid_query", "message": err.Error()}})
			return
		}
		page, err := l.List(c.Request.Context(), q)
		if err != nil {
			_ = c.Error(err)
			c.AbortWithStatusJSON(500, gin.H{"error": gin.H{"code": "internal_error", "message": "Не удалось загрузить журнал", "request_id": c.GetString("request_id")}})
			return
		}
		c.JSON(200, page)
	})
}
