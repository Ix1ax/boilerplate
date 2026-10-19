package events

import "github.com/gin-gonic/gin"

// RegisterDemandLive подключает подбор встреч и приватные отметки в день мероприятия.
func RegisterDemandLive(parent *gin.RouterGroup, s *Service) {
	store, ok := s.Store.(demandLiveStore)
	g := parent.Group("", profileSync(s), func(c *gin.Context) {
		if !ok {
			fail(c, 503, "unavailable", "Сервис временно недоступен")
			c.Abort()
			return
		}
		c.Next()
	})
	g.GET("/my/company-proposals", func(c *gin.Context) {
		items, err := store.DemandProposals(c.Request.Context(), actor(c))
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.POST("/company-proposals/:id/claim", withID(func(c *gin.Context, id string) {
		var in ProposalClaim
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Укажите точную точку сбора в поле location")
			return
		}
		e, err := store.ClaimDemandProposal(c.Request.Context(), actor(c), id, in)
		if handle(c, err) {
			return
		}
		c.JSON(201, e)
	}))
	g.POST("/company-proposals/:id/decline", withID(func(c *gin.Context, id string) {
		if handle(c, store.DeclineDemandProposal(c.Request.Context(), actor(c), id)) {
			return
		}
		c.Status(204)
	}))
	g.GET("/events/:id/arrivals", withID(func(c *gin.Context, id string) {
		items, err := store.Arrivals(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	}))
	g.PUT("/events/:id/arrival", withID(func(c *gin.Context, id string) {
		var in ArrivalInput
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Проверьте статус и время опоздания")
			return
		}
		item, err := store.SetArrival(c.Request.Context(), actor(c), id, in)
		if handle(c, err) {
			return
		}
		c.JSON(200, item)
	}))
}
