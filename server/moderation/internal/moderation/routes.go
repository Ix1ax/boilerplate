package moderation

import (
	"context"
	"time"

	"example.com/hackathon/server/moderation/internal/platform"
	"github.com/gin-gonic/gin"
)

// Reader — чтение проекций для админки (Repository).
type Reader interface {
	Dashboard(ctx context.Context) (Dashboard, error)
	Log(ctx context.Context, q platform.ListQuery) (Page[LogEntry], error)
	Comments(ctx context.Context, q platform.ListQuery) (Page[Comment], error)
	EventRows(ctx context.Context) ([][]string, error)
	OrganizationRows(ctx context.Context) ([][]string, error)
	LogRows(ctx context.Context) ([][]string, error)
}

// Русские названия статусов для отчётов: Excel читают люди, а не программы.
var statusNames = map[string]string{
	"pending": "на рассмотрении", "approved": "допущена", "rejected": "отклонена",
	"published": "опубликовано", "unpublished": "снято с публикации", "deleted": "удалено", "cancelled": "отменено",
	"visible": "видно", "hidden": "скрыто", "none": "—",
	"gathering": "набираем", "reached": "собран", "failed": "не собран",
	"event": "мероприятие", "organization": "организация", "comment": "сообщение",
}

func ru(v string) string {
	if n, ok := statusNames[v]; ok {
		return n
	}
	return v
}

// Register — маршруты админки под /admin/moderation (роль admin проверяют gateway и сервис).
//
//	GET /admin/moderation/dashboard              счётчики, ряд за 14 дней, топ направлений, давние заявки
//	GET /admin/moderation/log                    журнал решений: ?entity=&to=&entity_id=&q=&sort=
//	GET /admin/moderation/comments               лента сообщений всех обсуждений: ?status=&event_id=&q=
//	GET /admin/moderation/reports/events.csv     выгрузка мероприятий
//	GET /admin/moderation/reports/organizations.csv
//	GET /admin/moderation/reports/log.csv        журнал модерации
//
// Методы r вызываются только в обработчиках: тест сверки маршрутов передаёт nil.
func Register(group *gin.RouterGroup, r Reader) {
	g := group.Group("/admin/moderation", platform.RequireRole("admin"))
	g.GET("/dashboard", func(c *gin.Context) {
		d, err := r.Dashboard(c.Request.Context())
		if failed(c, err) {
			return
		}
		c.JSON(200, d)
	})
	g.GET("/log", list(logSpec, func(ctx context.Context, q platform.ListQuery) (Page[LogEntry], error) { return r.Log(ctx, q) }))
	g.GET("/comments", list(commentSpec, func(ctx context.Context, q platform.ListQuery) (Page[Comment], error) { return r.Comments(ctx, q) }))
	report := func(name string, header []string, load func(context.Context) ([][]string, error), statusCols ...int) gin.HandlerFunc {
		return func(c *gin.Context) {
			rows, err := load(c.Request.Context())
			if failed(c, err) {
				return
			}
			w := platform.NewCSV(c, name+"-"+time.Now().Format("2006-01-02")+".csv", header)
			for _, row := range rows {
				for _, i := range statusCols {
					row[i] = ru(row[i])
				}
				if err := w.Row(row...); err != nil {
					_ = c.Error(err)
					return
				}
			}
			if err := w.Flush(); err != nil {
				_ = c.Error(err)
			}
		}
	}
	g.GET("/reports/events.csv", report("events",
		[]string{"Название", "Направление", "Организатор", "Организация", "Начало (Красноярск)", "Место", "Запись", "Статус", "Причина", "Участников", "Заявок", "Новичков", "Мест", "Минимум", "Состав"},
		func(ctx context.Context) ([][]string, error) { return r.EventRows(ctx) }, 7, 14))
	g.GET("/reports/organizations.csv", report("organizations",
		[]string{"Название", "Город", "Представитель", "Статус", "Причина отказа", "Подана", "Событий"},
		func(ctx context.Context) ([][]string, error) { return r.OrganizationRows(ctx) }, 3))
	g.GET("/reports/log.csv", report("moderation-log",
		[]string{"Когда (Красноярск)", "Объект", "Название", "Было", "Стало", "Кто", "Причина"},
		func(ctx context.Context) ([][]string, error) { return r.LogRows(ctx) }, 1, 3, 4))
}

func list[T any](spec platform.ListSpec, load func(context.Context, platform.ListQuery) (Page[T], error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		q, err := platform.ParseList(c, spec, 1)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"code": "invalid_query", "message": err.Error(), "request_id": c.GetString("request_id")}})
			return
		}
		p, err := load(c.Request.Context(), q)
		if failed(c, err) {
			return
		}
		c.JSON(200, p)
	}
}

func failed(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	_ = c.Error(err)
	c.AbortWithStatusJSON(500, gin.H{"error": gin.H{"code": "internal_error", "message": "Не удалось загрузить данные админки", "request_id": c.GetString("request_id")}})
	return true
}
