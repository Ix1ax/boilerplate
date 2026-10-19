package accounts

import (
	"errors"
	"net/url"
	"strconv"
	"time"

	"example.com/hackathon/server/identity/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RegisterInternal — служебные вызовы gateway (группа уже проверяет INTERNAL_TOKEN).
// Браузер сюда не попадает: gateway проксирует только /api/v1/admin/accounts.
//
//	POST /internal/v1/accounts                регистрация
//	POST /internal/v1/login                   проверка почты и пароля
//	GET  /internal/v1/accounts/:id            актуальный статус для проверки сессии
//	POST /internal/v1/accounts/:id/password   смена пароля с проверкой текущего
func RegisterInternal(g *gin.RouterGroup, s *Service) {
	g.POST("/accounts", func(c *gin.Context) {
		var in RegisterInput
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полями email, password и name")
			return
		}
		a, err := s.Register(c.Request.Context(), in)
		if handle(c, err) {
			return
		}
		c.JSON(201, a)
	})
	g.POST("/login", func(c *gin.Context) {
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
	g.GET("/accounts/:id", withID(func(c *gin.Context, id string) {
		a, err := s.Get(c.Request.Context(), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, a)
	}))
	g.POST("/accounts/:id/password", withID(func(c *gin.Context, id string) {
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

// RegisterAdmin — учётные записи обычной регистрации в админке. Роль admin проверяется
// и в gateway, и здесь.
//
//	GET  /admin/accounts?q=&status=       поиск по почте и имени
//	GET  /admin/accounts/:id              карточка с историей
//	POST /admin/accounts/:id/block        заблокировать с причиной
//	POST /admin/accounts/:id/unblock      разблокировать
func RegisterAdmin(api *gin.RouterGroup, s *Service) {
	admin := api.Group("/admin/accounts", platform.RequireRole("admin"))
	admin.GET("", func(c *gin.Context) {
		limit, offset, ok := platform.ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
		status := c.Query("status")
		if !ok || (status != "" && status != "active" && status != "blocked") || len([]rune(c.Query("q"))) > 100 {
			fail(c, 400, "invalid_query", "Проверьте status (active, blocked), q (до 100 символов), limit и offset")
			return
		}
		page, err := s.List(c.Request.Context(), c.Query("q"), status, limit, offset)
		if handle(c, err) {
			return
		}
		c.JSON(200, page)
	})
	admin.GET("/:id", withID(func(c *gin.Context, id string) {
		d, err := s.Detail(c.Request.Context(), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, d)
	}))
	admin.POST("/:id/block", withID(func(c *gin.Context, id string) {
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
	admin.POST("/:id/unblock", withID(func(c *gin.Context, id string) {
		a, err := s.Unblock(c.Request.Context(), id, actorLabel(c))
		if handle(c, err) {
			return
		}
		c.JSON(200, a)
	}))
}

// actorLabel — кто принял решение: имя из gateway (URL-кодировано) и id для однозначности.
func actorLabel(c *gin.Context) string {
	name, _ := url.PathUnescape(c.GetHeader("X-User-Name"))
	if name == "" {
		return platform.Owner(c)
	}
	return name + " (" + platform.Owner(c) + ")"
}

func withID(h func(c *gin.Context, id string)) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if uuid.Validate(id) != nil {
			fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
			return
		}
		h(c, id)
	}
}

// handle переводит ошибки в коды ответа. Сообщения доменных ошибок можно показывать людям.
func handle(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var fe FieldError
	var blocked BlockedError
	var locked LockedError
	switch {
	case errors.As(err, &fe):
		c.AbortWithStatusJSON(422, gin.H{"error": gin.H{"code": "validation_error", "message": fe.Message, "field": fe.Field, "request_id": c.GetString("request_id")}})
	case errors.As(err, &blocked):
		c.AbortWithStatusJSON(403, gin.H{"error": gin.H{"code": "account_blocked", "message": blocked.Error(), "reason": blocked.Reason, "request_id": c.GetString("request_id")}})
	case errors.As(err, &locked):
		c.Header("Retry-After", strconv.Itoa(int(time.Until(locked.Until).Seconds())+1))
		fail(c, 429, "too_many_attempts", locked.Error())
	case errors.Is(err, ErrInvalidCredentials):
		fail(c, 401, "invalid_credentials", err.Error())
	case errors.Is(err, ErrWrongPassword):
		c.AbortWithStatusJSON(422, gin.H{"error": gin.H{"code": "validation_error", "message": err.Error(), "field": "current_password", "request_id": c.GetString("request_id")}})
	case errors.Is(err, ErrSamePassword):
		c.AbortWithStatusJSON(422, gin.H{"error": gin.H{"code": "validation_error", "message": err.Error(), "field": "new_password", "request_id": c.GetString("request_id")}})
	case errors.Is(err, ErrEmailTaken):
		fail(c, 409, "email_taken", err.Error())
	case errors.Is(err, ErrNotFound):
		fail(c, 404, "not_found", err.Error())
	case errors.Is(err, ErrTransition):
		fail(c, 409, "invalid_transition", err.Error())
	case errors.Is(err, ErrReasonRequired):
		fail(c, 422, "reason_required", err.Error())
	default:
		_ = c.Error(err)
		fail(c, 500, "internal_error", "Не удалось выполнить действие. Попробуйте ещё раз")
	}
	return true
}

func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
