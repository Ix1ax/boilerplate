package tasks

import (
	"context"
	"time"

	"example.com/hackathon/server/tasks/internal/platform"
	"github.com/gin-gonic/gin"
)

// Stats — данные для карточек и графика дашборда.
type Stats struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	Owners     int `json:"owners"`
	Moderation struct {
		OK          int `json:"ok"`
		NeedsReview int `json:"needs_review"`
		Unchecked   int `json:"unchecked"`
	} `json:"moderation"`
	ByDay []DayCount `json:"by_day"`
}
type DayCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// Exporter — чтение всех задач для отчётов (только для администратора).
type Exporter interface {
	ExportAll(ctx context.Context, row func(owner string, t Task) error) error
	Stats(ctx context.Context) (Stats, error)
}

// RegisterAdmin добавляет админские маршруты. Образец отчётной выгрузки для любого домена.
func RegisterAdmin(group *gin.RouterGroup, e Exporter) {
	admin := group.Group("/admin", platform.RequireRole("admin"))
	admin.GET("/tasks/stats", func(c *gin.Context) {
		st, err := e.Stats(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			c.AbortWithStatusJSON(500, gin.H{"error": gin.H{"code": "internal_error", "message": "Не удалось посчитать статистику", "request_id": c.GetString("request_id")}})
			return
		}
		c.JSON(200, st)
	})
	admin.GET("/tasks/export.csv", func(c *gin.Context) {
		out := platform.NewCSV(c, "tasks-"+time.Now().Format("2006-01-02")+".csv",
			[]string{"ID", "Владелец", "Название", "Статус", "Создана", "Проверка ИИ", "Источник проверки"})
		err := e.ExportAll(c.Request.Context(), func(owner string, t Task) error {
			verdict, source := "не проверена", ""
			if t.Moderation != nil {
				verdict, source = map[string]string{"ok": "замечаний нет", "needs_review": "нужна проверка"}[t.Moderation.Verdict], t.Moderation.Source
			}
			return out.Row(t.ID, owner, t.Title, t.Status, t.CreatedAt.Format(time.RFC3339), verdict, source)
		})
		if err != nil {
			// Заголовки уже отправлены: обрываем файл и пишем причину в лог.
			_ = c.Error(err)
			return
		}
		_ = out.Flush()
	})
}
