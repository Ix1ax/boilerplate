package events

import (
	"errors"
	"strings"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Register — маршруты пользователя, организатора и представителя организации.
// Личность берётся из заголовков gateway, а не из тела запроса.
//
//	GET    /sports                                       справочник направлений
//	GET    /events                                       выдача с фильтрами и поиском
//	POST   /events                                       создать (от себя или от допущенной организации)
//	GET    /events/:id                                   карточка
//	POST   /events/:id/cancel                            отмена организатором
//	POST   /events/:id/join, DELETE /events/:id/join     записаться или подать заявку / отменить
//	GET    /events/:id/participants                      участники и заявки
//	POST   /events/:id/participants/:pid/approve|reject  решение по заявке
//	GET    /events/:id/comments, POST …                  обсуждение
//	GET    /my/participations, /my/events                мои записи и мои события
//	GET    /my/organization, POST, PUT                   заявка организации
//	GET    /organizations, /organizations/:id            каталог и страница организации
//	GET    /profiles/:id                                 профиль организатора
//	… /my/company-requests, /sports/demand               «Ищу компанию» (company_routes.go)
//	… /events/:id/buddy, /remind, /weather              проводники, напоминания, погода (gathering_routes.go)
func Register(parent *gin.RouterGroup, s *Service) {
	g := parent.Group("", profileSync(s))
	g.GET("/sports", func(c *gin.Context) {
		items, err := s.Sports(c.Request.Context())
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.GET("/events", func(c *gin.Context) {
		f, err := parseFilter(c)
		if err != nil {
			fail(c, 400, "invalid_query", err.Error())
			return
		}
		page, err := s.ListEvents(c.Request.Context(), actor(c), f)
		if handle(c, err) {
			return
		}
		c.JSON(200, page)
	})
	g.POST("/events", func(c *gin.Context) {
		var in EventInput
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Проверьте формат данных: дата — в формате RFC 3339")
			return
		}
		e, err := s.CreateEvent(c.Request.Context(), actor(c), in)
		if handle(c, err) {
			return
		}
		c.JSON(201, e)
	})
	g.GET("/events/:id", withID(func(c *gin.Context, id string) {
		e, err := s.GetEvent(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, e)
	}))
	g.POST("/events/:id/cancel", withID(func(c *gin.Context, id string) {
		var in struct {
			Reason string `json:"reason"`
		}
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полем reason")
			return
		}
		e, err := s.CancelEvent(c.Request.Context(), actor(c), id, in.Reason)
		if handle(c, err) {
			return
		}
		c.JSON(200, e)
	}))
	g.POST("/events/:id/join", withID(func(c *gin.Context, id string) {
		var in struct {
			Message   string `json:"message"`
			FirstTime bool   `json:"first_time"`
		}
		// Тело необязательно: для свободной записи сообщение не нужно.
		if c.Request.ContentLength > 0 && c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полями message и first_time")
			return
		}
		p, err := s.Join(c.Request.Context(), actor(c), id, in.Message, in.FirstTime)
		if handle(c, err) {
			return
		}
		c.JSON(200, p)
	}))
	g.DELETE("/events/:id/join", withID(func(c *gin.Context, id string) {
		if handle(c, s.Leave(c.Request.Context(), actor(c), id)) {
			return
		}
		c.Status(204)
	}))
	g.GET("/events/:id/participants", withID(func(c *gin.Context, id string) {
		items, err := s.Participants(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	}))
	decide := func(approve bool) gin.HandlerFunc {
		return withID(func(c *gin.Context, id string) {
			pid := c.Param("pid")
			if uuid.Validate(pid) != nil {
				fail(c, 400, "invalid_id", "Ожидается идентификатор UUID")
				return
			}
			var in struct {
				Reason string `json:"reason"`
			}
			if c.Request.ContentLength > 0 && c.ShouldBindJSON(&in) != nil {
				fail(c, 400, "invalid_body", "Ожидается JSON с полем reason")
				return
			}
			p, err := s.Decide(c.Request.Context(), actor(c), id, pid, approve, in.Reason)
			if handle(c, err) {
				return
			}
			c.JSON(200, p)
		})
	}
	g.POST("/events/:id/participants/:pid/approve", decide(true))
	g.POST("/events/:id/participants/:pid/reject", decide(false))
	g.GET("/events/:id/comments", withID(func(c *gin.Context, id string) {
		items, err := s.Comments(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	}))
	g.POST("/events/:id/comments", withID(func(c *gin.Context, id string) {
		var in struct {
			Body     string  `json:"body"`
			ParentID *string `json:"parent_id"`
		}
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полем body")
			return
		}
		cm, err := s.AddComment(c.Request.Context(), actor(c), id, in.ParentID, in.Body)
		if handle(c, err) {
			return
		}
		c.JSON(201, cm)
	}))
	g.GET("/my/participations", func(c *gin.Context) {
		items, err := s.MyParticipations(c.Request.Context(), actor(c))
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.GET("/my/events", func(c *gin.Context) {
		a := actor(c)
		page, err := s.ListEvents(c.Request.Context(), a, EventFilter{OrganizerID: a.ID, Statuses: []string{"published", "unpublished", "deleted", "cancelled"}, Limit: 100})
		if handle(c, err) {
			return
		}
		c.JSON(200, page)
	})
	g.GET("/my/organization", func(c *gin.Context) {
		o, err := s.MyOrganization(c.Request.Context(), actor(c))
		if handle(c, err) {
			return
		}
		c.JSON(200, o)
	})
	orgInput := func(c *gin.Context) (OrganizationInput, bool) {
		var in OrganizationInput
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Ожидается JSON с полями name, description, city, contacts, sport_ids")
			return in, false
		}
		return in, true
	}
	g.POST("/my/organization", func(c *gin.Context) {
		in, ok := orgInput(c)
		if !ok {
			return
		}
		o, err := s.SubmitOrganization(c.Request.Context(), actor(c), in)
		if handle(c, err) {
			return
		}
		c.JSON(201, o)
	})
	g.PUT("/my/organization", func(c *gin.Context) {
		in, ok := orgInput(c)
		if !ok {
			return
		}
		o, err := s.UpdateOrganization(c.Request.Context(), actor(c), in)
		if handle(c, err) {
			return
		}
		c.JSON(200, o)
	})
	g.GET("/organizations", func(c *gin.Context) {
		limit, offset, ok := platform.ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
		if !ok {
			fail(c, 400, "invalid_query", "limit должен быть от 1 до 100, offset — от 0 до 100000")
			return
		}
		page, err := s.ListOrganizations(c.Request.Context(), actor(c), c.Query("q"), nil, limit, offset)
		if handle(c, err) {
			return
		}
		c.JSON(200, page)
	})
	g.GET("/organizations/:id", withID(func(c *gin.Context, id string) {
		a := actor(c)
		o, err := s.GetOrganization(c.Request.Context(), a, id)
		if handle(c, err) {
			return
		}
		// Страница организации показывает её ближайшие события (требование кейса).
		page, err := s.ListEvents(c.Request.Context(), a, EventFilter{OrganizationID: id, Limit: 50})
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"organization": o, "events": page.Items})
	}))
	g.GET("/profiles/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "" || len(id) > 100 {
			fail(c, 400, "invalid_id", "Некорректный идентификатор пользователя")
			return
		}
		p, err := s.GetProfile(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, p)
	})
	registerCompany(g, s)
	registerGatherings(g, s)
}

// RegisterAdmin — модерация. Роль admin проверяется и в gateway, и здесь.
//
//	GET  /admin/organizations?status=&q=           очередь заявок и поиск организаций
//	POST /admin/organizations/:id/approve|reject   допуск (причина обязательна при отказе)
//	GET  /admin/events?status=&q=                  поиск событий во всех статусах
//	POST /admin/events/:id/unpublish|delete        снять или удалить с причиной
//	POST /admin/events/:id/restore                 вернуть в публикацию
//	POST /admin/comments/:id/hide                  скрыть сообщение с причиной
//	… карточки, история, справочник, переотправка    admin.go (registerAdminExtra)
func RegisterAdmin(g *gin.RouterGroup, s *Service) {
	admin := g.Group("/admin", platform.RequireRole("admin"), profileSync(s))
	admin.GET("/organizations", func(c *gin.Context) {
		limit, offset, ok := platform.ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
		statuses, valid := parseStatuses(c.Query("status"), map[string]bool{"pending": true, "approved": true, "rejected": true})
		if !ok || !valid {
			fail(c, 400, "invalid_query", "Проверьте status (pending, approved, rejected), limit и offset")
			return
		}
		page, err := s.ListOrganizations(c.Request.Context(), actor(c), c.Query("q"), statuses, limit, offset)
		if handle(c, err) {
			return
		}
		c.JSON(200, page)
	})
	review := func(to string) gin.HandlerFunc {
		return withID(func(c *gin.Context, id string) {
			reason, ok := reasonBody(c)
			if !ok {
				return
			}
			o, err := s.ReviewOrganization(c.Request.Context(), actor(c), id, to, reason)
			if handle(c, err) {
				return
			}
			c.JSON(200, o)
		})
	}
	admin.POST("/organizations/:id/approve", review("approved"))
	admin.POST("/organizations/:id/reject", review("rejected"))
	admin.GET("/events", func(c *gin.Context) {
		f, err := parseFilter(c)
		statuses, valid := parseStatuses(c.Query("status"), map[string]bool{"published": true, "unpublished": true, "deleted": true, "cancelled": true})
		if err != nil || !valid {
			fail(c, 400, "invalid_query", "Проверьте параметры: status — published, unpublished, deleted, cancelled")
			return
		}
		if len(statuses) == 0 {
			statuses = []string{"published", "unpublished", "deleted", "cancelled"}
		}
		f.Statuses = statuses
		page, err := s.ListEvents(c.Request.Context(), actor(c), f)
		if handle(c, err) {
			return
		}
		c.JSON(200, page)
	})
	moderate := func(to string) gin.HandlerFunc {
		return withID(func(c *gin.Context, id string) {
			reason, ok := reasonBody(c)
			if !ok {
				return
			}
			e, err := s.ModerateEvent(c.Request.Context(), actor(c), id, to, reason)
			if handle(c, err) {
				return
			}
			c.JSON(200, e)
		})
	}
	admin.POST("/events/:id/unpublish", moderate("unpublished"))
	admin.POST("/events/:id/delete", moderate("deleted"))
	admin.POST("/events/:id/restore", moderate("published"))
	admin.POST("/comments/:id/hide", withID(func(c *gin.Context, id string) {
		reason, ok := reasonBody(c)
		if !ok {
			return
		}
		cm, err := s.HideComment(c.Request.Context(), actor(c), id, reason)
		if handle(c, err) {
			return
		}
		c.JSON(200, cm)
	}))
	registerAdminExtra(admin, s)
}

func actor(c *gin.Context) Actor {
	return Actor{ID: platform.Owner(c), Name: platform.UserName(c), Email: platform.Email(c.Request.Context()), Admin: platform.HasRole(c, "admin")}
}

// profileSync обновляет копию имени пользователя. Сбой не мешает запросу: имя подтянется позже.
func profileSync(s *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := s.EnsureProfile(c.Request.Context(), actor(c)); err != nil {
			_ = c.Error(err)
		}
		c.Next()
	}
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

func reasonBody(c *gin.Context) (string, bool) {
	var in struct {
		Reason string `json:"reason"`
	}
	if c.Request.ContentLength > 0 && c.ShouldBindJSON(&in) != nil {
		fail(c, 400, "invalid_body", "Ожидается JSON с полем reason")
		return "", false
	}
	return in.Reason, true
}

func parseStatuses(raw string, allowed map[string]bool) ([]string, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, true
	}
	out := strings.Split(raw, ",")
	for _, s := range out {
		if !allowed[s] {
			return nil, false
		}
	}
	return out, true
}

func parseFilter(c *gin.Context) (EventFilter, error) {
	limit, offset, ok := platform.ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
	if !ok {
		return EventFilter{}, errors.New("limit должен быть от 1 до 100, offset — от 0 до 100000")
	}
	f := EventFilter{Limit: limit, Offset: offset, Query: c.Query("q"), Level: c.Query("level"), Mode: c.Query("mode"), OrganizationID: c.Query("organization")}
	if len([]rune(f.Query)) > 100 {
		return f, errors.New("Поисковый запрос длиннее 100 символов")
	}
	if f.Level != "" && !levels[f.Level] {
		return f, errors.New("level: any, beginner, amateur или pro")
	}
	if f.Mode != "" && f.Mode != "open" && f.Mode != "request" {
		return f, errors.New("mode: open или request")
	}
	if f.OrganizationID != "" && uuid.Validate(f.OrganizationID) != nil {
		return f, errors.New("organization: ожидается UUID")
	}
	if raw := strings.TrimSpace(c.Query("sport")); raw != "" {
		f.SportIDs = strings.Split(raw, ",")
		for _, id := range f.SportIDs {
			if uuid.Validate(id) != nil || len(f.SportIDs) > 20 {
				return f, errors.New("sport: список UUID через запятую, не больше 20")
			}
		}
	}
	for param, dst := range map[string]**time.Time{"from": &f.From, "to": &f.To} {
		if raw := c.Query(param); raw != "" {
			t, err := parseTime(raw)
			if err != nil {
				return f, errors.New(param + ": ожидается дата 2026-10-20 или время в RFC 3339")
			}
			*dst = &t
		}
	}
	return f, nil
}

func parseTime(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	return time.Parse(time.DateOnly, raw)
}

// handle переводит ошибки домена в коды ответа. Сообщения домена — для людей, их можно показывать.
func handle(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var fe FieldError
	switch {
	case errors.As(err, &fe):
		c.AbortWithStatusJSON(422, gin.H{"error": gin.H{"code": "validation_error", "message": fe.Message, "field": fe.Field, "request_id": c.GetString("request_id")}})
	case errors.Is(err, ErrNotFound):
		fail(c, 404, "not_found", "Не найдено или недоступно")
	case errors.Is(err, ErrOrgNotApproved), errors.Is(err, ErrNotOrganizer):
		fail(c, 403, "forbidden", err.Error())
	case errors.Is(err, ErrEventClosed), errors.Is(err, ErrEventPast), errors.Is(err, ErrOwnEvent),
		errors.Is(err, ErrFull), errors.Is(err, ErrOrgExists), errors.Is(err, ErrRequestExists),
		errors.Is(err, ErrNotConfirmed), errors.Is(err, ErrNewcomerBuddy), errors.Is(err, ErrRemindedRecently), errors.Is(err, ErrBaseSport), errors.Is(err, ErrNotParticipant), errors.Is(err, ErrSportHidden):
		fail(c, 409, "conflict", err.Error())
	case errors.Is(err, ErrBadParent):
		fail(c, 422, "validation_error", err.Error())
	case errors.Is(err, platform.ErrTransition):
		fail(c, 409, "invalid_transition", err.Error())
	case errors.Is(err, platform.ErrCommentRequired):
		fail(c, 422, "reason_required", "Укажите причину: её увидит другая сторона")
	default:
		_ = c.Error(err)
		fail(c, 500, "internal_error", "Не удалось выполнить действие. Попробуйте ещё раз")
	}
	return true
}

func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
