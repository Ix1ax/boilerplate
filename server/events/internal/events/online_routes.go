package events

import (
	"errors"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterOnline(g *gin.RouterGroup, s *Service) {
	config := VideoConfig{os.Getenv("LIVEKIT_URL"), os.Getenv("LIVEKIT_API_KEY"), os.Getenv("LIVEKIT_API_SECRET")}
	g.GET("/events/:id/format", withID(func(c *gin.Context, id string) {
		store, ok := s.Store.(OnlineStore)
		if !ok {
			fail(c, 503, "unavailable", "Форматы встреч недоступны")
			return
		}
		out, err := store.GetEventFormat(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, out)
	}))
	g.PUT("/events/:id/format", withID(func(c *gin.Context, id string) {
		var in struct {
			Format string `json:"format"`
		}
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Укажите формат встречи")
			return
		}
		store, ok := s.Store.(OnlineStore)
		if !ok {
			fail(c, 503, "unavailable", "Форматы встреч недоступны")
			return
		}
		out, err := store.SetEventFormat(c.Request.Context(), actor(c), id, in.Format)
		if handle(c, err) {
			return
		}
		c.JSON(200, out)
	}))
	g.POST("/events/:id/video/token", withID(func(c *gin.Context, id string) {
		c.Header("Cache-Control", "no-store")
		store, ok := s.Store.(OnlineStore)
		if !ok {
			fail(c, 503, "unavailable", "Видеовстречи недоступны")
			return
		}
		a := actor(c)
		v, err := store.GetVideoAccess(c.Request.Context(), a, id)
		if handle(c, err) {
			return
		}
		out, err := IssueVideoToken(config, a, v, time.Now().UTC())
		switch {
		case errors.Is(err, ErrVideoUnavailable):
			fail(c, 503, "video_unavailable", err.Error())
			return
		case errors.Is(err, ErrVideoWindow), errors.Is(err, ErrVideoOffline):
			fail(c, 409, "video_unavailable", err.Error())
			return
		}
		if handle(c, err) {
			return
		}
		c.JSON(200, out)
	}))
}
