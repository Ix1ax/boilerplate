package accounts

import (
	"example.com/hackathon/server/identity/internal/platform"
	"github.com/gin-gonic/gin"
)

// RegisterAdminInternal — вход администраторов для gateway (группа уже проверяет INTERNAL_TOKEN).
//
//	POST /internal/v1/admins/login            проверка почты и пароля администратора
//	GET  /internal/v1/admins/:id              актуальный статус для проверки админской сессии
//	POST /internal/v1/admins/:id/password     смена своего пароля с проверкой текущего
func RegisterAdminInternal(g *gin.RouterGroup, s *Admins) {
	g.POST("/admins/login", func(c *gin.Context) {
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полями email и password")
			return
		}
		a, err := s.Login(c.Request.Context(), in.Email, in.Password)
		if handle(c, err) {
			return
		}
		c.JSON(200, a)
	})
	g.GET("/admins/:id", withID(func(c *gin.Context, id string) {
		a, err := s.Get(c.Request.Context(), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, a)
	}))
	g.POST("/admins/:id/password", withID(func(c *gin.Context, id string) {
		var in struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полями current_password и new_password")
			return
		}
		if handle(c, s.ChangePassword(c.Request.Context(), id, in.CurrentPassword, in.NewPassword)) {
			return
		}
		c.Status(204)
	}))
}

// RegisterAdmins — управление администраторами. Только суперадминистратор: роль superadmin
// выставляет gateway из админской сессии, проверка повторяется здесь.
//
//	GET    /admin/admins?q=&status=      список и поиск (удалённые — только status=deleted)
//	POST   /admin/admins                 создать администратора
//	GET    /admin/admins/:id             карточка с историей
//	POST   /admin/admins/:id/block       заблокировать с причиной
//	POST   /admin/admins/:id/unblock     разблокировать
//	POST   /admin/admins/:id/password    задать новый пароль (сессии админа закрываются)
//	DELETE /admin/admins/:id             удалить (мягко: история остаётся)
func RegisterAdmins(api *gin.RouterGroup, s *Admins) {
	g := api.Group("/admin/admins", platform.RequireRole(RoleSuperadmin))
	g.GET("", func(c *gin.Context) {
		limit, offset, ok := platform.ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
		status := c.Query("status")
		if !ok || (status != "" && status != "active" && status != "blocked" && status != "deleted") || len([]rune(c.Query("q"))) > 100 {
			fail(c, 400, "invalid_query", "Проверьте status (active, blocked, deleted), q (до 100 символов), limit и offset")
			return
		}
		page, err := s.List(c.Request.Context(), c.Query("q"), status, limit, offset)
		if handle(c, err) {
			return
		}
		c.JSON(200, page)
	})
	g.POST("", func(c *gin.Context) {
		var in AdminInput
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полями email, name и password")
			return
		}
		a, err := s.Create(c.Request.Context(), in, platform.Owner(c), actorLabel(c))
		if handle(c, err) {
			return
		}
		c.JSON(201, a)
	})
	g.GET("/:id", withID(func(c *gin.Context, id string) {
		d, err := s.Detail(c.Request.Context(), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, d)
	}))
	g.POST("/:id/block", withID(func(c *gin.Context, id string) {
		var in struct {
			Reason string `json:"reason"`
		}
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полем reason")
			return
		}
		a, err := s.Block(c.Request.Context(), id, in.Reason, actorLabel(c))
		if handle(c, err) {
			return
		}
		c.JSON(200, a)
	}))
	g.POST("/:id/unblock", withID(func(c *gin.Context, id string) {
		a, err := s.Unblock(c.Request.Context(), id, actorLabel(c))
		if handle(c, err) {
			return
		}
		c.JSON(200, a)
	}))
	g.POST("/:id/password", withID(func(c *gin.Context, id string) {
		var in struct {
			Password string `json:"password"`
		}
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полем password")
			return
		}
		a, err := s.ResetPassword(c.Request.Context(), id, in.Password, actorLabel(c))
		if handle(c, err) {
			return
		}
		c.JSON(200, a)
	}))
	g.DELETE("/:id", withID(func(c *gin.Context, id string) {
		a, err := s.Delete(c.Request.Context(), id, actorLabel(c))
		if handle(c, err) {
			return
		}
		c.JSON(200, a)
	}))
}
