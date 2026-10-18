// Package files — приватные файлы: метаданные в Postgres, содержимое в S3. Скачать и удалить файл
// может только владелец; файл можно привязать к записи другого сервиса (Entity).
package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MaxSize = 10 << 20

var allowedContentTypes = map[string]bool{
	"image/png":                 true,
	"image/jpeg":                true,
	"image/webp":                true,
	"application/pdf":           true,
	"text/plain; charset=utf-8": true,
}

var (
	ErrNotFound           = errors.New("Файл не найден")
	ErrInvalidSize        = errors.New("Размер файла должен быть от 1 байта до 10 МиБ")
	ErrUnsupportedType    = errors.New("Можно загрузить PNG, JPEG, WebP, PDF или текстовый файл")
	ErrInvalidName        = errors.New("Недопустимое имя файла")
	ErrStorageUnavailable = errors.New("Хранилище временно недоступно")
	ErrInvalidEntity      = errors.New("entity_type — латиница и «_» до 30 символов, entity_id — до 64 символов из букв, цифр и «-»")
)

// File — метаданные файла. Само содержимое лежит в S3 под ключом <владелец>/<id>.
type File struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int       `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
	// Entity — к какой записи другого сервиса приложен файл (документы заявки и т.п.); nil — без привязки.
	Entity *Entity `json:"entity"`
}

// Entity — ссылка на запись другого сервиса. Files не проверяет её существование: доступ к файлу
// всё равно ограничен владельцем, а проверку «можно ли прикладывать» делает доменный сервис.
type Entity struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

var (
	entityTypeRe = regexp.MustCompile(`^[a-z][a-z_]{0,29}$`)
	entityIDRe   = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
)

// ParseEntity: обе части пустые — без привязки; заполнена одна или формат неверен — ErrInvalidEntity.
func ParseEntity(entityType, entityID string) (*Entity, error) {
	if entityType == "" && entityID == "" {
		return nil, nil
	}
	if !entityTypeRe.MatchString(entityType) || !entityIDRe.MatchString(entityID) {
		return nil, ErrInvalidEntity
	}
	return &Entity{Type: entityType, ID: entityID}, nil
}

type Page struct {
	Items  []File `json:"items"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// Store — метаданные файлов (Postgres).
type Store interface {
	// entity != nil — только файлы, приложенные к этой записи.
	List(ctx context.Context, owner string, entity *Entity, limit, offset int) (Page, error)
	Insert(ctx context.Context, owner, objectKey string, f File) error
	Get(ctx context.Context, owner, id string) (objectKey, name, contentType string, err error)
	Delete(ctx context.Context, owner, id string) error
}

// Blobs — бинарное содержимое (S3-совместимое хранилище). Отдельный порт от
// Store: подменяется независимо (другой бакет, другой провайдер, мок в тестах).
type Blobs interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Get(ctx context.Context, key string) (body io.ReadCloser, size int64, err error)
	// Удаление отсутствующего объекта успешно: это позволяет повторить операцию после сбоя БД.
	Delete(ctx context.Context, key string) error
}

// Service — операции с файлами владельца. Доступ сначала проверяется в Postgres, затем идёт работа с S3.
type Service struct {
	store Store
	blobs Blobs
}

func New(store Store, blobs Blobs) *Service { return &Service{store: store, blobs: blobs} }

func (s *Service) List(ctx context.Context, owner string, entity *Entity, limit, offset int) (Page, error) {
	return s.store.List(ctx, owner, entity, limit, offset)
}

// Upload загружает файл без привязки к записи (см. UploadFor).
func (s *Service) Upload(ctx context.Context, owner, name string, data []byte) (File, error) {
	return s.UploadFor(ctx, owner, name, data, nil)
}

// UploadFor загружает файл с привязкой к записи (entity может быть nil).
func (s *Service) UploadFor(ctx context.Context, owner, name string, data []byte, entity *Entity) (File, error) {
	if len(data) == 0 || len(data) > MaxSize {
		return File{}, ErrInvalidSize
	}
	// Расширение и Content-Type от клиента недостоверны: проверяем содержимое файла.
	contentType := http.DetectContentType(data)
	if !allowedContentTypes[contentType] {
		return File{}, ErrUnsupportedType
	}
	if name == "" || len(name) > 255 || strings.ContainsAny(name, "\r\n") {
		return File{}, ErrInvalidName
	}
	f := File{ID: uuid.NewString(), Name: name, Size: len(data), ContentType: contentType, CreatedAt: time.Now().UTC(), Entity: entity}
	key := owner + "/" + f.ID
	if err := s.blobs.Put(ctx, key, contentType, data); err != nil {
		return File{}, fmt.Errorf("%w: %w", ErrStorageUnavailable, err)
	}
	// У S3 и PostgreSQL нет общей транзакции: при отказе БД пытаемся убрать загруженный объект.
	if err := s.store.Insert(ctx, owner, key, f); err != nil {
		_ = s.blobs.Delete(ctx, key)
		return File{}, err
	}
	return f, nil
}

func (s *Service) Download(ctx context.Context, owner, id string) (body io.ReadCloser, size int64, name, contentType string, err error) {
	key, name, contentType, err := s.store.Get(ctx, owner, id)
	if err != nil {
		return nil, 0, "", "", err
	}
	body, size, err = s.blobs.Get(ctx, key)
	if err != nil {
		return nil, 0, "", "", fmt.Errorf("%w: %w", ErrStorageUnavailable, err)
	}
	return body, size, name, contentType, nil
}

func (s *Service) Delete(ctx context.Context, owner, id string) error {
	// Владелец проверяется в SQL до обращения к S3, а не по данным из браузера.
	key, _, _, err := s.store.Get(ctx, owner, id)
	if err != nil {
		return err
	}
	if err := s.blobs.Delete(ctx, key); err != nil {
		return fmt.Errorf("%w: %w", ErrStorageUnavailable, err)
	}
	// Метаданные сохраняются до успеха S3, чтобы удаление можно было повторить.
	return s.store.Delete(ctx, owner, id)
}
