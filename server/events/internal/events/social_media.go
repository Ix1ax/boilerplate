package events

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const storyMaxMedia = 10 << 20

var ErrStoryStorage = errors.New("Хранилище историй временно недоступно")

// StoryBlobs хранит только объекты events. Метаданные файлов другого сервиса не читаем.
type StoryBlobs interface {
	Put(context.Context, string, string, []byte) error
	Get(context.Context, string) (io.ReadCloser, int64, error)
	Delete(context.Context, string) error
}
type StoryMediaInfo struct {
	Key, ContentType string
	Size             int64
}

func (r *Repository) ReserveStoryMedia(ctx context.Context, a Actor, id string) (string, error) {
	if err := socialUUID(id); err != nil {
		return "", err
	}
	key := "events-stories/" + uuid.NewString()
	tag, err := r.db.Exec(ctx, `INSERT INTO story_media_objects(object_key,story_id) SELECT $1,id FROM stories WHERE id=$2 AND author_id=$3 AND expires_at>now() AND deleted_at IS NULL AND media_key IS NULL`, key, id, a.ID)
	if err == nil && tag.RowsAffected() == 0 {
		return "", ErrNotFound
	}
	return key, err
}
func (r *Repository) AttachStoryMedia(ctx context.Context, a Actor, id, key, contentType string, size int64) error {
	tag, err := r.db.Exec(ctx, `UPDATE stories SET media_key=$3,media_content_type=$4,media_size=$5 WHERE id=$1 AND author_id=$2
AND expires_at>now() AND deleted_at IS NULL AND media_key IS NULL AND EXISTS(SELECT 1 FROM story_media_objects WHERE object_key=$3 AND story_id=$1 AND created_at>now()-interval '1 hour')`, id, a.ID, key, contentType, size)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrSocialConflict
	}
	return err
}
func (r *Repository) StoryMediaAccess(ctx context.Context, a Actor, id string) (StoryMediaInfo, error) {
	var out StoryMediaInfo
	if err := socialUUID(id); err != nil {
		return out, err
	}
	err := r.db.QueryRow(ctx, `SELECT s.media_key,s.media_content_type,s.media_size FROM stories s WHERE s.id=$1 AND s.media_key IS NOT NULL
AND s.deleted_at IS NULL AND s.expires_at>now() AND (s.author_id=$2 OR EXISTS(SELECT 1 FROM friendships f
WHERE f.user_a=LEAST($2,s.author_id) AND f.user_b=GREATEST($2,s.author_id) AND f.status='accepted'))`, id, a.ID).Scan(&out.Key, &out.ContentType, &out.Size)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return out, err
}

// CleanupStoryMedia повторяется после сбоя S3 или БД. Удаление S3 идемпотентно;
// ссылка на живую историю исключена из выборки, незавершённым загрузкам даём час.
func (r *Repository) CleanupStoryMedia(ctx context.Context, blobs StoryBlobs) error {
	if blobs == nil {
		return nil
	}
	rows, err := r.db.Query(ctx, `SELECT o.object_key FROM story_media_objects o JOIN stories s ON s.id=o.story_id
WHERE s.expires_at<=now() OR s.deleted_at IS NOT NULL OR (o.created_at<now()-interval '1 hour' AND s.media_key IS DISTINCT FROM o.object_key)
ORDER BY o.created_at LIMIT 100`)
	if err != nil {
		return err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err = blobs.Delete(ctx, key); err != nil {
			return err
		}
		if _, err = r.db.Exec(ctx, `DELETE FROM story_media_objects WHERE object_key=$1`, key); err != nil {
			return err
		}
	}
	return nil
}
func registerStoryMedia(g *gin.RouterGroup, s *Service, store SocialStore) {
	g.POST("/stories/:story/media", func(c *gin.Context) {
		if s.StoryMedia == nil {
			fail(c, 503, "storage_unavailable", ErrStoryStorage.Error())
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, storyMaxMedia+(64<<10))
		if err := c.Request.ParseMultipartForm(storyMaxMedia); err != nil {
			fail(c, 413, "payload_too_large", "Файл превышает 10 МиБ или неверный формат загрузки")
			return
		}
		defer c.Request.MultipartForm.RemoveAll()
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			fail(c, 400, "invalid_body", "Выберите файл в поле file")
			return
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, storyMaxMedia+1))
		if err != nil || len(data) == 0 || len(data) > storyMaxMedia {
			fail(c, 413, "payload_too_large", "Размер файла — от 1 байта до 10 МиБ")
			return
		}
		contentType := http.DetectContentType(data)
		switch contentType {
		case "image/jpeg", "image/png", "image/webp", "image/gif", "video/mp4", "video/webm":
		default:
			fail(c, 415, "unsupported_type", "Допустимы JPEG, PNG, WebP, GIF, MP4 и WebM")
			return
		}
		key, err := store.ReserveStoryMedia(c.Request.Context(), actor(c), c.Param("story"))
		if handle(c, err) {
			return
		}
		if err = s.StoryMedia.Put(c.Request.Context(), key, contentType, data); err != nil {
			_ = c.Error(err)
			fail(c, 503, "storage_unavailable", ErrStoryStorage.Error())
			return
		}
		err = store.AttachStoryMedia(c.Request.Context(), actor(c), c.Param("story"), key, contentType, int64(len(data)))
		if err != nil {
			socialReply(c, 201, nil, err)
			return
		}
		c.JSON(201, gin.H{"media_url": "/api/v1/stories/" + c.Param("story") + "/media", "content_type": contentType, "size": len(data)})
	})
	g.GET("/stories/:story/media", func(c *gin.Context) {
		info, err := store.StoryMediaAccess(c.Request.Context(), actor(c), c.Param("story"))
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
		// Приватный ответ нельзя сохранять в общем кеше: дружба и срок истории могут измениться.
		c.Header("Cache-Control", "private, no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Disposition", "inline")
		c.DataFromReader(200, size, info.ContentType, io.LimitReader(body, storyMaxMedia), nil)
	})
}
