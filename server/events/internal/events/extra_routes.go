package events

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// registerExtras — профиль, правка события, «найти другую встречу», календарь.
//
//	GET   /my/profile                 мой профиль со статистикой организатора
//	PATCH /my/profile                 о себе, город, уровень
//	PATCH /events/:id                 правка организатором; перенос — с уведомлением «было → стало»
//	GET   /events/:id/alternatives    похожие предстоящие встречи (?limit=1..20)
//	GET   /events/:id/calendar.ics    событие для календаря
func registerExtras(g *gin.RouterGroup, s *Service) {
	g.GET("/my/profile", func(c *gin.Context) {
		a := actor(c)
		p, err := s.GetProfile(c.Request.Context(), a, a.ID)
		if handle(c, err) {
			return
		}
		c.JSON(200, p)
	})
	g.PATCH("/my/profile", func(c *gin.Context) {
		var in ProfilePatch
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полями about, city, level")
			return
		}
		p, err := s.UpdateMyProfile(c.Request.Context(), actor(c), in)
		if handle(c, err) {
			return
		}
		c.JSON(200, p)
	})
	g.PATCH("/events/:id", withID(func(c *gin.Context, id string) {
		var in EventPatch
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Проверьте формат данных: дата — в формате RFC 3339")
			return
		}
		e, err := s.UpdateEvent(c.Request.Context(), actor(c), id, in)
		if handle(c, err) {
			return
		}
		c.JSON(200, e)
	}))
	g.GET("/events/:id/alternatives", withID(func(c *gin.Context, id string) {
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "5"))
		if err != nil || limit < 1 || limit > 20 {
			fail(c, 400, "invalid_query", "limit — от 1 до 20")
			return
		}
		items, err := s.Alternatives(c.Request.Context(), actor(c), id, limit)
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	}))
	g.GET("/events/:id/calendar.ics", withID(func(c *gin.Context, id string) {
		e, err := s.GetEvent(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.Header("Content-Disposition", `attachment; filename="event.ics"`)
		c.Data(200, "text/calendar; charset=utf-8", []byte(ICS(e, time.Now())))
	}))
}
