package events

import (
	"context"
	"errors"
	"fmt"
	"io"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Модерация соцконтента по жалобам: отзывы, истории и личные сообщения. Переписку админ не
// читает: он видит только сообщение, на которое пожаловались, и может скрыть только его.

// socialModerationStore — операции админа над соцконтентом (их реализует Repository).
type socialModerationStore interface {
	HideReview(ctx context.Context, a Actor, id, reason string) error
	DeleteStory(ctx context.Context, a Actor, id, reason string) error
	HideDirectMessage(ctx context.Context, a Actor, id, reason string) error
	AdminStoryMediaAccess(ctx context.Context, a Actor, id string) (StoryMediaInfo, error)
}

// HideDirectMessage — админ скрывает личное сообщение. Только то, на которое есть жалоба: без
// жалобы админ о сообщении не знает, и доступ к чужой переписке «на всякий случай» закрыт в SQL.
func (r *Repository) HideDirectMessage(ctx context.Context, a Actor, id, reason string) error {
	if err := socialUUID(id); err != nil {
		return err
	}
	reason, err := socialBody(reason, 500)
	if err != nil {
		return err
	}
	return r.inTx(ctx, func(tx pgx.Tx) error {
		var chatID, author, peer string
		var hidden bool
		err := tx.QueryRow(ctx, `SELECT m.chat_id::text,m.author_id,CASE WHEN c.user_a=m.author_id THEN c.user_b ELSE c.user_a END,m.hidden_at IS NOT NULL
FROM direct_messages m JOIN direct_chats c ON c.id=m.chat_id
WHERE m.id=$1 AND $2 AND EXISTS(SELECT 1 FROM reports x WHERE x.target_type='direct_message' AND x.target_id=m.id)
FOR UPDATE OF m`, id, a.Admin).Scan(&chatID, &author, &peer, &hidden)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if hidden {
			return fmt.Errorf("%w: сообщение уже скрыто", platform.ErrTransition)
		}
		if _, err = tx.Exec(ctx, `UPDATE direct_messages SET hidden_at=now(),hidden_by=$2,hidden_reason=$3 WHERE id=$1`, id, a.ID, reason); err != nil {
			return err
		}
		if err = platform.RecordTransition(ctx, tx, "direct_message", id, platform.Transition{From: "visible", To: "hidden", Actor: actorLabel(a), Comment: reason}); err != nil {
			return err
		}
		if err = closeReports(ctx, tx, a, "direct_message", id, "resolved", reason); err != nil {
			return err
		}
		if err = notify(ctx, tx, author, "Ваше личное сообщение скрыто модератором", reason, ""); err != nil {
			return err
		}
		for _, u := range []string{author, peer} {
			if err = socialSignal(ctx, tx, u, "chat.changed", chatID); err != nil {
				return err
			}
		}
		return nil
	})
}

// AdminStoryMediaAccess — фото или видео истории для разбора жалобы: без проверки дружбы и срока,
// но только у истории, на которую жаловались.
func (r *Repository) AdminStoryMediaAccess(ctx context.Context, a Actor, id string) (StoryMediaInfo, error) {
	var out StoryMediaInfo
	if err := socialUUID(id); err != nil {
		return out, err
	}
	err := r.db.QueryRow(ctx, `SELECT s.media_key,s.media_content_type,s.media_size FROM stories s WHERE s.id=$1 AND s.media_key IS NOT NULL
AND $2 AND EXISTS(SELECT 1 FROM reports x WHERE x.target_type='story' AND x.target_id=s.id)`, id, a.Admin).Scan(&out.Key, &out.ContentType, &out.Size)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return out, err
}

// registerAdminSocial — решения админа по жалобам на соцконтент (группа уже проверяет admin).
//
//	POST /admin/reviews/:id/hide       скрыть отзыв {reason}
//	POST /admin/stories/:id/delete     удалить историю {reason}
//	GET  /admin/stories/:id/media      фото или видео истории из жалобы
//	POST /admin/messages/:id/hide      скрыть личное сообщение из жалобы {reason}
//
// Каждое решение закрывает открытые жалобы на цель и уведомляет её автора.
func registerAdminSocial(admin *gin.RouterGroup, s *Service) {
	store, ok := s.Store.(socialModerationStore)
	decide := func(op func(socialModerationStore, context.Context, Actor, string, string) error) gin.HandlerFunc {
		return withID(func(c *gin.Context, id string) {
			if !ok {
				fail(c, 503, "unavailable", "Сервис временно недоступен")
				return
			}
			reason, valid := reasonBody(c)
			if !valid {
				return
			}
			if reason == "" {
				fail(c, 422, "validation_error", "Укажите причину: её увидит автор")
				return
			}
			if handle(c, op(store, c.Request.Context(), actor(c), id, reason)) {
				return
			}
			c.Status(204)
		})
	}
	admin.POST("/reviews/:id/hide", decide(socialModerationStore.HideReview))
	admin.POST("/stories/:id/delete", decide(socialModerationStore.DeleteStory))
	admin.POST("/messages/:id/hide", decide(socialModerationStore.HideDirectMessage))
	admin.GET("/stories/:id/media", withID(func(c *gin.Context, id string) {
		if !ok {
			fail(c, 503, "unavailable", "Сервис временно недоступен")
			return
		}
		info, err := store.AdminStoryMediaAccess(c.Request.Context(), actor(c), id)
		if handle(c, err) {
			return
		}
		if s.StoryMedia == nil {
			fail(c, 503, "storage_unavailable", ErrStoryStorage.Error())
			return
		}
		body, size, err := s.StoryMedia.Get(c.Request.Context(), info.Key)
		if err != nil {
			_ = c.Error(err)
			fail(c, 503, "storage_unavailable", ErrStoryStorage.Error())
			return
		}
		defer body.Close()
		if size != info.Size || size < 1 || size > storyMaxMedia {
			fail(c, 503, "storage_unavailable", ErrStoryStorage.Error())
			return
		}
		c.Header("Cache-Control", "private, no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Disposition", "inline")
		c.DataFromReader(200, size, info.ContentType, io.LimitReader(body, storyMaxMedia), nil)
	}))
}
