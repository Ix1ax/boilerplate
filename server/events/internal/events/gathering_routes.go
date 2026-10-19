package events

import "github.com/gin-gonic/gin"

// registerGatherings — «Встречу новичка», напоминания и погода.
//
//	POST   /events/:id/buddy    готов встретить новичка (подтверждённый участник, не новичок)
//	DELETE /events/:id/buddy    передумал: подопечные переходят к другим проводникам
//	POST   /events/:id/remind   организатор: напомнить подтверждённым сейчас (не чаще раза в час)
//	GET    /events/:id/weather  прогноз для уличной встречи с координатами
func registerGatherings(g *gin.RouterGroup, s *Service) {
	buddy := func(on bool) gin.HandlerFunc {
		return withID(func(c *gin.Context, id string) {
			p, err := s.SetBuddy(c.Request.Context(), actor(c), id, on)
			if handle(c, err) {
				return
			}
			c.JSON(200, p)
		})
	}
	g.POST("/events/:id/buddy", buddy(true))
	g.DELETE("/events/:id/buddy", buddy(false))
	g.POST("/events/:id/remind", withID(func(c *gin.Context, id string) {
		n, err := s.Remind(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"sent": n})
	}))
	g.GET("/events/:id/weather", withID(func(c *gin.Context, id string) {
		w, err := s.EventWeather(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, w)
	}))
}
