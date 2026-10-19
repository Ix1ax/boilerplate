package events

import (
	"example.com/hackathon/server/events/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// registerReports — жалобы, удаление своего сообщения и исключение участника.
//
//	POST   /events/:id/report                       пожаловаться на мероприятие
//	POST   /events/:id/comments/:cid/report         пожаловаться на сообщение
//	POST   /reviews/:review/report                  пожаловаться на отзыв
//	POST   /stories/:story/report                   пожаловаться на историю (видна жалующемуся)
//	POST   /chats/:chat/messages/:message/report    пожаловаться на личное сообщение (участник чата)
//	DELETE /events/:id/comments/:cid                удалить своё сообщение
//	GET    /my/reports                              мои жалобы и решения по ним
//	POST   /events/:id/participants/:pid/remove     организатор исключает участника с причиной
func registerReports(g *gin.RouterGroup, s *Service) {
	report := func(targetType string) gin.HandlerFunc {
		return withID(func(c *gin.Context, id string) {
			target := id
			if targetType == "comment" {
				if target = c.Param("cid"); uuid.Validate(target) != nil {
					fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
					return
				}
			}
			var in ReportInput
			if c.ShouldBindJSON(&in) != nil {
				fail(c, 400, "invalid_body", "Ожидается JSON с полями reason и details")
				return
			}
			r, err := s.Report(c.Request.Context(), actor(c), targetType, id, target, in)
			if handle(c, err) {
				return
			}
			c.JSON(201, r)
		})
	}
	g.POST("/events/:id/report", report("event"))
	g.POST("/events/:id/comments/:cid/report", report("comment"))
	// Имена параметров совпадают с соседними маршрутами соцфункций: так требует дерево gin.
	reportSocial := func(targetType, parentParam, idParam string) gin.HandlerFunc {
		return func(c *gin.Context) {
			parent, target := "", c.Param(idParam)
			if parentParam != "" {
				parent = c.Param(parentParam)
			}
			if uuid.Validate(target) != nil || parentParam != "" && uuid.Validate(parent) != nil {
				fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
				return
			}
			var in ReportInput
			if c.ShouldBindJSON(&in) != nil {
				fail(c, 400, "invalid_body", "Ожидается JSON с полями reason и details")
				return
			}
			r, err := s.Report(c.Request.Context(), actor(c), targetType, parent, target, in)
			if handle(c, err) {
				return
			}
			c.JSON(201, r)
		}
	}
	g.POST("/reviews/:review/report", reportSocial("review", "", "review"))
	g.POST("/stories/:story/report", reportSocial("story", "", "story"))
	g.POST("/chats/:chat/messages/:message/report", reportSocial("direct_message", "chat", "message"))
	g.DELETE("/events/:id/comments/:cid", withID(func(c *gin.Context, id string) {
		cid := c.Param("cid")
		if uuid.Validate(cid) != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		cm, err := s.DeleteComment(c.Request.Context(), actor(c), id, cid)
		if handle(c, err) {
			return
		}
		c.JSON(200, cm)
	}))
	g.GET("/my/reports", func(c *gin.Context) {
		items, err := s.MyReports(c.Request.Context(), actor(c))
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.POST("/events/:id/participants/:pid/remove", withID(func(c *gin.Context, id string) {
		pid := c.Param("pid")
		if uuid.Validate(pid) != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		reason, ok := reasonBody(c)
		if !ok {
			return
		}
		p, err := s.RemoveParticipant(c.Request.Context(), actor(c), id, pid, reason)
		if handle(c, err) {
			return
		}
		c.JSON(200, p)
	}))
}

// registerAdminReports — очередь жалоб и возврат скрытых сообщений (группа уже проверяет admin).
//
//	GET  /admin/reports?status=&target_type=   очередь: открытые и самые частые цели первыми
//	POST /admin/reports/:id/dismiss            отклонить все открытые жалобы на эту цель с пояснением
//	POST /admin/comments/:id/restore           вернуть скрытое сообщение
//
// Принять жалобу — значит скрыть или удалить цель обычными маршрутами модерации (сообщение,
// событие, а для соцконтента — registerAdminSocial): открытые жалобы закрываются в той же транзакции.
func registerAdminReports(admin *gin.RouterGroup, s *Service) {
	admin.GET("/reports", func(c *gin.Context) {
		limit, offset, ok := platform.ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
		status, target := c.Query("status"), c.Query("target_type")
		if !ok || (status != "" && status != "open" && status != "resolved" && status != "dismissed") || (target != "" && !reportTargetTypes[target]) {
			fail(c, 400, "invalid_query", "Проверьте status (open, resolved, dismissed), target_type (event, comment, review, story, direct_message), limit и offset")
			return
		}
		page, err := s.AdminReports(c.Request.Context(), status, target, limit, offset)
		if handle(c, err) {
			return
		}
		c.JSON(200, page)
	})
	admin.POST("/reports/:id/dismiss", withID(func(c *gin.Context, id string) {
		var in struct {
			Resolution string `json:"resolution"`
		}
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полем resolution")
			return
		}
		r, err := s.DismissReport(c.Request.Context(), actor(c), id, in.Resolution)
		if handle(c, err) {
			return
		}
		c.JSON(200, r)
	}))
	admin.POST("/comments/:id/restore", withID(func(c *gin.Context, id string) {
		cm, err := s.RestoreComment(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, cm)
	}))
}
