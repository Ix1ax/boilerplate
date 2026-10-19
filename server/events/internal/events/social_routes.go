package events

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RegisterSocial подключает социальные возможности к проверенной gateway личности.
// Чаты мероприятий используют /events/:id/comments: второй копии сообщений нет.
func RegisterSocial(parent *gin.RouterGroup, s *Service) {
	store, _ := s.Store.(SocialStore)
	g := parent.Group("", profileSync(s), func(c *gin.Context) { c.Header("Cache-Control", "private, no-store"); c.Next() })
	registerStoryMedia(g, s, store)
	g.GET("/my/friends", func(c *gin.Context) {
		p, ok := socialPagination(c, false)
		if !ok {
			return
		}
		out, err := store.Friends(c.Request.Context(), actor(c), p)
		socialReply(c, 200, out, err)
	})
	for _, action := range []string{"request", "accept", "decline", "remove"} {
		action := action
		g.POST("/friends/:uid/"+action, func(c *gin.Context) {
			out, err := store.Friendship(c.Request.Context(), actor(c), c.Param("uid"), action)
			socialReply(c, 200, out, err)
		})
	}
	g.POST("/chats", func(c *gin.Context) {
		var in struct {
			UserID string `json:"user_id"`
		}
		if !socialBind(c, &in) {
			return
		}
		out, err := store.OpenDirectChat(c.Request.Context(), actor(c), in.UserID)
		socialReply(c, 200, out, err)
	})
	g.GET("/my/chats", func(c *gin.Context) {
		p, ok := socialPagination(c, true)
		if !ok {
			return
		}
		out, err := store.Chats(c.Request.Context(), actor(c), p)
		socialReply(c, 200, out, err)
	})
	g.GET("/chats/:chat/messages", func(c *gin.Context) {
		p, ok := socialPagination(c, true)
		if !ok {
			return
		}
		out, err := store.DirectMessages(c.Request.Context(), actor(c), c.Param("chat"), p)
		socialReply(c, 200, out, err)
	})
	g.POST("/chats/:chat/messages", func(c *gin.Context) {
		var in struct {
			Body string `json:"body"`
		}
		if !socialBind(c, &in) {
			return
		}
		out, err := store.SendDirectMessage(c.Request.Context(), actor(c), c.Param("chat"), in.Body)
		socialReply(c, 201, out, err)
	})
	g.POST("/events/:id/reviews", func(c *gin.Context) {
		var in struct {
			Score int    `json:"score"`
			Body  string `json:"body"`
		}
		if !socialBind(c, &in) {
			return
		}
		out, err := store.CreateReview(c.Request.Context(), actor(c), c.Param("id"), in.Score, in.Body)
		socialReply(c, 201, out, err)
	})
	g.GET("/profiles/:id/reviews", func(c *gin.Context) {
		p, ok := socialPagination(c, true)
		if !ok {
			return
		}
		out, err := store.ProfileReviews(c.Request.Context(), actor(c), c.Param("id"), p)
		socialReply(c, 200, out, err)
	})
	g.POST("/reviews/:review/hide", func(c *gin.Context) {
		var in struct {
			Reason string `json:"reason"`
		}
		if !socialBind(c, &in) {
			return
		}
		err := store.HideReview(c.Request.Context(), actor(c), c.Param("review"), in.Reason)
		socialReply(c, 204, nil, err)
	})
	g.POST("/stories", func(c *gin.Context) {
		var in struct {
			Body string `json:"body"`
		}
		if !socialBind(c, &in) {
			return
		}
		out, err := store.CreateStory(c.Request.Context(), actor(c), in.Body)
		socialReply(c, 201, out, err)
	})
	g.GET("/stories", func(c *gin.Context) {
		p, ok := socialPagination(c, true)
		if !ok {
			return
		}
		out, err := store.Stories(c.Request.Context(), actor(c), p)
		socialReply(c, 200, out, err)
	})
	g.POST("/stories/:story/delete", func(c *gin.Context) {
		var in struct {
			Reason string `json:"reason"`
		}
		if !socialBind(c, &in) {
			return
		}
		err := store.DeleteStory(c.Request.Context(), actor(c), c.Param("story"), in.Reason)
		socialReply(c, 204, nil, err)
	})
	g.POST("/organizations/:id/membership", func(c *gin.Context) {
		err := store.ClubMembership(c.Request.Context(), actor(c), c.Param("id"), true)
		socialReply(c, 204, nil, err)
	})
	g.DELETE("/organizations/:id/membership", func(c *gin.Context) {
		err := store.ClubMembership(c.Request.Context(), actor(c), c.Param("id"), false)
		socialReply(c, 204, nil, err)
	})
	g.GET("/organizations/:id/members", func(c *gin.Context) {
		p, ok := socialPagination(c, false)
		if !ok {
			return
		}
		out, err := store.ClubMembers(c.Request.Context(), actor(c), c.Param("id"), p)
		socialReply(c, 200, out, err)
	})
}
func socialBind(c *gin.Context, in any) bool {
	if c.ShouldBindJSON(in) != nil {
		fail(c, 400, "invalid_body", "Проверьте формат JSON")
		return false
	}
	return true
}
func socialPagination(c *gin.Context, uuidCursor bool) (SocialPage, bool) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "30"))
	if err != nil {
		fail(c, 422, "validation_error", "Размер страницы — от 1 до 100")
		return SocialPage{}, false
	}
	p, err := parseSocialPage(c.Query("cursor"), limit)
	if err == nil && uuidCursor && uuid.Validate(p.ID) != nil {
		err = FieldError{"cursor", "Некорректный курсор"}
	}
	if handle(c, err) {
		return p, false
	}
	return p, true
}
func socialReply(c *gin.Context, status int, out any, err error) {
	if errors.Is(err, ErrSocialConflict) {
		fail(c, 409, "social_conflict", err.Error())
		return
	}
	if handle(c, err) {
		return
	}
	if status == 204 {
		c.Status(status)
	} else {
		c.JSON(status, out)
	}
}
