package tasks

import (
	"context"
	"strconv"
	"time"

	"example.com/hackathon/apps/tasks/internal/platform"
	"github.com/gin-gonic/gin"
)

// Exporter — чтение всех задач для отчёта (только для администратора).
type Exporter interface {
	ExportAll(ctx context.Context, row func(owner string, t Task) error) error
}

// RegisterAdmin добавляет админские маршруты. Образец отчётной выгрузки для любого домена.
func RegisterAdmin(group *gin.RouterGroup, e Exporter) {
	admin := group.Group("/admin", platform.RequireRole("admin"))
	admin.GET("/tasks/export.csv", func(c *gin.Context) {
		out := platform.NewCSV(c, "tasks-"+time.Now().Format("2006-01-02")+".csv",
			[]string{"ID", "Владелец", "Название", "Выполнена", "Создана", "Проверка ИИ", "Источник проверки"})
		err := e.ExportAll(c.Request.Context(), func(owner string, t Task) error {
			verdict, source := "не проверена", ""
			if t.Moderation != nil {
				verdict, source = map[string]string{"ok": "замечаний нет", "needs_review": "нужна проверка"}[t.Moderation.Verdict], t.Moderation.Source
			}
			return out.Row(t.ID, owner, t.Title, strconv.FormatBool(t.Completed), t.CreatedAt.Format(time.RFC3339), verdict, source)
		})
		if err != nil {
			// Заголовки уже отправлены: обрываем файл и пишем причину в лог.
			_ = c.Error(err)
			return
		}
		_ = out.Flush()
	})
}
