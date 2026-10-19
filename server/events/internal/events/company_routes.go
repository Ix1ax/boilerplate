package events

import (
	"example.com/hackathon/server/events/internal/platform"
	"github.com/gin-gonic/gin"
)

// registerCompany — маршруты «Ищу компанию».
//
//	GET    /my/company-requests              мои запросы (активные первыми)
//	POST   /my/company-requests              создать; сразу подбираются уже опубликованные события
//	GET    /my/company-requests/:id          один запрос
//	DELETE /my/company-requests/:id          закрыть: «нашёл компанию» или «больше не ищу»
//	GET    /my/company-requests/:id/matches  подобранные предстоящие события
//	GET    /sports/demand?sport_id=|sport_name=&level=  обезличенный спрос для организатора
func registerCompany(g *gin.RouterGroup, s *Service) {
	g.GET("/my/company-requests", func(c *gin.Context) {
		items, err := s.MyCompanyRequests(c.Request.Context(), actor(c))
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.POST("/my/company-requests", func(c *gin.Context) {
		var in CompanyRequestInput
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Проверьте формат данных")
			return
		}
		r, err := s.CreateCompanyRequest(c.Request.Context(), actor(c), in)
		if handle(c, err) {
			return
		}
		c.JSON(201, r)
	})
	g.GET("/my/company-requests/:id", withID(func(c *gin.Context, id string) {
		r, err := s.MyCompanyRequest(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, r)
	}))
	g.DELETE("/my/company-requests/:id", withID(func(c *gin.Context, id string) {
		if handle(c, s.CloseCompanyRequest(c.Request.Context(), actor(c), id)) {
			return
		}
		c.Status(204)
	}))
	g.GET("/my/company-requests/:id/matches", withID(func(c *gin.Context, id string) {
		limit, offset, ok := platform.ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
		if !ok {
			fail(c, 400, "invalid_query", "limit должен быть от 1 до 100, offset — от 0 до 100000")
			return
		}
		a := actor(c)
		// Сначала сам запрос: чужой или несуществующий — 404, а не пустой список.
		if _, err := s.MyCompanyRequest(c.Request.Context(), a, id); handle(c, err) {
			return
		}
		page, err := s.ListEvents(c.Request.Context(), a, EventFilter{MatchedFor: id, Limit: limit, Offset: offset})
		if handle(c, err) {
			return
		}
		c.JSON(200, page)
	}))
	g.GET("/sports/demand", func(c *gin.Context) {
		d, err := s.Demand(c.Request.Context(), actor(c), c.Query("sport_id"), c.Query("sport_name"), c.Query("level"))
		if handle(c, err) {
			return
		}
		c.JSON(200, d)
	})
}
