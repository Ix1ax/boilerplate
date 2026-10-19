package events

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Административные операции над доменом: карточки с историей решений и модерация справочника.
// Аналитика, журнал модерации и отчёты — в отдельном сервисе moderation, он строит их из доменных
// событий (domain.go). Здесь — только то, что читает или меняет данные домена.

// OrganizationDetail — карточка организации для админа: история допуска и все её события.
type OrganizationDetail struct {
	Organization
	History []platform.Transition `json:"history"`
	Events  []Event               `json:"events"`
}

// HistoryEntry — переход события или его состава («Соберём состав»): kind различает их в ленте.
type HistoryEntry struct {
	Kind string `json:"kind"`
	platform.Transition
}

// AdminSport — направление справочника с тем, кто его добавил и где оно используется.
type AdminSport struct {
	Sport
	Hidden        bool      `json:"hidden"`
	CreatedBy     string    `json:"created_by"`
	EventsCount   int       `json:"events_count"`
	RequestsCount int       `json:"requests_count"`
	CreatedAt     time.Time `json:"created_at"`
}

var ErrBaseSport = errors.New("Базовые направления справочника не скрываются — только добавленные пользователями")

func (r *Repository) AdminOrganization(ctx context.Context, a Actor, id string) (OrganizationDetail, error) {
	o, err := r.GetOrganization(ctx, a, id)
	if err != nil {
		return OrganizationDetail{}, err
	}
	history, err := platform.History(ctx, r.db, "organization", id)
	if err != nil {
		return OrganizationDetail{}, err
	}
	page, err := r.ListEvents(ctx, a, EventFilter{OrganizationID: id, Statuses: []string{"published", "unpublished", "deleted", "cancelled"}, Limit: 100})
	if err != nil {
		return OrganizationDetail{}, err
	}
	return OrganizationDetail{Organization: o, History: history, Events: page.Items}, nil
}

func (r *Repository) EventHistory(ctx context.Context, a Actor, id string) ([]HistoryEntry, error) {
	if _, err := r.GetEvent(ctx, a, id); err != nil {
		return nil, err
	}
	out := []HistoryEntry{}
	for _, kind := range []string{"event", "event_quorum"} {
		h, err := platform.History(ctx, r.db, kind, id)
		if err != nil {
			return nil, err
		}
		for _, t := range h {
			out = append(out, HistoryEntry{Kind: kind, Transition: t})
		}
	}
	slices.SortStableFunc(out, func(x, y HistoryEntry) int { return x.CreatedAt.Compare(y.CreatedAt) })
	return out, nil
}

// AdminSports — весь справочник, пользовательские и скрытые первыми: модератору важны именно они.
func (r *Repository) AdminSports(ctx context.Context) ([]AdminSport, error) {
	rows, err := r.db.Query(ctx, `SELECT s.id, s.name, s.is_custom, s.hidden, COALESCE(pr.display_name,''),
(SELECT count(*) FROM events e WHERE e.sport_id=s.id),
(SELECT count(*) FROM company_requests q WHERE q.sport_id=s.id AND q.status='active'),
s.created_at
FROM sports s LEFT JOIN profiles pr ON pr.user_id=s.created_by
ORDER BY s.is_custom DESC, s.hidden DESC, s.created_at DESC, lower(s.name)`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (AdminSport, error) {
		var s AdminSport
		return s, row.Scan(&s.ID, &s.Name, &s.IsCustom, &s.Hidden, &s.CreatedBy, &s.EventsCount, &s.RequestsCount, &s.CreatedAt)
	})
}

// SetSportHidden скрывает пользовательское направление (спам, оскорбление, дубль) или возвращает его.
// Уже созданные события не трогаем: их снимают отдельно; новые с этим направлением создать нельзя.
func (r *Repository) SetSportHidden(ctx context.Context, a Actor, id string, hidden bool, reason string) (AdminSport, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var custom, was bool
		err := tx.QueryRow(ctx, "SELECT is_custom, hidden FROM sports WHERE id=$1 FOR UPDATE", id).Scan(&custom, &was)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrNotFound
		case err != nil:
			return err
		case !custom:
			return ErrBaseSport
		case was == hidden:
			return fmt.Errorf("%w: направление уже в этом состоянии", platform.ErrTransition)
		case hidden && reason == "":
			return platform.ErrCommentRequired
		}
		if _, err = tx.Exec(ctx, "UPDATE sports SET hidden=$2 WHERE id=$1", id, hidden); err != nil {
			return err
		}
		from, to := "visible", "hidden"
		if !hidden {
			from, to = to, from
		}
		return platform.RecordTransition(ctx, tx, "sport", id, platform.Transition{From: from, To: to, Actor: actorLabel(a), Comment: reason})
	})
	if err != nil {
		return AdminSport{}, err
	}
	all, err := r.AdminSports(ctx)
	if err != nil {
		return AdminSport{}, err
	}
	for _, s := range all {
		if s.ID == id {
			return s, nil
		}
	}
	return AdminSport{}, ErrNotFound
}

// registerAdminExtra — дополнения админки к RegisterAdmin (группа уже проверяет роль admin).
//
//	GET  /admin/organizations/:id    карточка: история допуска и все события организации
//	GET  /admin/events/:id/history   история публикации и состава события
//	GET  /admin/sports               справочник с пользовательскими направлениями и их использованием
//	POST /admin/sports/:id/hide|show скрыть пользовательское направление с причиной / вернуть
//	POST /admin/domain/republish     переотправить снимки в moderation (догнать после очистки)
func registerAdminExtra(admin *gin.RouterGroup, s *Service) {
	admin.GET("/organizations/:id", withID(func(c *gin.Context, id string) {
		o, err := s.AdminOrganization(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, o)
	}))
	admin.GET("/events/:id/history", withID(func(c *gin.Context, id string) {
		h, err := s.EventHistory(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": h})
	}))
	admin.GET("/sports", func(c *gin.Context) {
		items, err := s.AdminSports(c.Request.Context())
		if handle(c, err) {
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	sport := func(hidden bool) gin.HandlerFunc {
		return withID(func(c *gin.Context, id string) {
			reason := ""
			if hidden {
				var ok bool
				if reason, ok = reasonBody(c); !ok {
					return
				}
			}
			reason, err := cleanReason(reason)
			if handle(c, err) {
				return
			}
			sp, err := s.SetSportHidden(c.Request.Context(), actor(c), id, hidden, reason)
			if handle(c, err) {
				return
			}
			c.JSON(200, sp)
		})
	}
	admin.POST("/sports/:id/hide", sport(true))
	admin.POST("/sports/:id/show", sport(false))
	admin.POST("/domain/republish", func(c *gin.Context) {
		n, err := s.RepublishDomain(c.Request.Context())
		if handle(c, err) {
			return
		}
		c.JSON(202, gin.H{"queued": n})
	})
}
