// Package notifications — внутренние уведомления пользователя. Сервис не знает предметных
// областей: другие сервисы публикуют событие notification.requested, а он сохраняет и отдаёт его.
package notifications

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrNotFound    = errors.New("Уведомление не найдено")
	ErrMaxDisabled = errors.New("Уведомления в MAX не настроены")
)

type Notification struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Link      string     `json:"link"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at"`
}
type Page struct {
	Items  []Notification `json:"items"`
	Total  int            `json:"total"`
	Unread int            `json:"unread"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type Store interface {
	List(ctx context.Context, user string, limit, offset int) (Page, error)
	// Insert возвращает false, если событие уже обработано (повторная доставка).
	Insert(ctx context.Context, user, eventID string, n Notification) (bool, error)
	MarkRead(ctx context.Context, user, id string) error
	MarkAllRead(ctx context.Context, user string) error
}

type Service struct {
	store    Store
	hub      *Hub
	mailer   Mailer
	log      func(msg string, args ...any)
	max      MaxChannel
	maxLinks MaxStore
}

// MaxChannel — мессенджер MAX (maxbot.Bot). nil — канал выключен (нет токена бота).
type MaxChannel interface {
	Send(ctx context.Context, maxUserID int64, text string) error
	LinkURL(code string) string
}

// MaxStore — привязки аккаунтов к MAX (Repository).
type MaxStore interface {
	CreateLinkCode(ctx context.Context, user, code string, ttl time.Duration) error
	MaxUserID(ctx context.Context, user string) (int64, bool, error)
	UnlinkUser(ctx context.Context, user string) error
}

// WithMax включает канал MAX.
func (s *Service) WithMax(ch MaxChannel, store MaxStore) *Service {
	s.max, s.maxLinks = ch, store
	return s
}

// New: hub и mailer необязательны (nil — без live-событий и писем).
func New(store Store, hub *Hub, mailer Mailer, log func(msg string, args ...any)) *Service {
	if log == nil {
		log = func(string, ...any) {}
	}
	return &Service{store: store, hub: hub, mailer: mailer, log: log}
}

func (s *Service) Hub() *Hub { return s.hub }

func (s *Service) List(ctx context.Context, user string, limit, offset int) (Page, error) {
	return s.store.List(ctx, user, limit, offset)
}
func (s *Service) MarkRead(ctx context.Context, user, id string) error {
	return s.store.MarkRead(ctx, user, id)
}
func (s *Service) MarkAllRead(ctx context.Context, user string) error {
	return s.store.MarkAllRead(ctx, user)
}

// HandleEvent принимает notification.requested. Невалидные данные отбрасываются (не повторяются):
// ошибка издателя не должна зацикливать очередь. Ссылка допускается только внутренняя.
func (s *Service) HandleEvent(ctx context.Context, eventID, eventType, userID string, data []byte) error {
	if eventType != "notification.requested" || userID == "" {
		return nil
	}
	var in struct{ Title, Body, Link, Email string }
	if json.Unmarshal(data, &in) != nil {
		return nil
	}
	in.Title, in.Body = strings.TrimSpace(in.Title), strings.TrimSpace(in.Body)
	if in.Title == "" || utf8.RuneCountInString(in.Title) > 200 || utf8.RuneCountInString(in.Body) > 1000 {
		return nil
	}
	if !strings.HasPrefix(in.Link, "/") || strings.HasPrefix(in.Link, "//") {
		in.Link = ""
	}
	if _, err := uuid.Parse(eventID); err != nil {
		eventID = uuid.NewString()
	}
	n := Notification{ID: uuid.NewString(), Title: in.Title, Body: in.Body, Link: in.Link, CreatedAt: time.Now().UTC()}
	created, err := s.store.Insert(ctx, userID, eventID, n)
	if err != nil || !created {
		return err
	}
	if s.hub != nil {
		s.hub.Publish(userID, n)
	}
	// Письмо — дополнительный канал: его сбой логируем, но не повторяем доставку события,
	// иначе пользователь получит дубли писем.
	if s.mailer != nil && in.Email != "" {
		if err := s.mailer.Send(ctx, in.Email, n.Title, n.Body); err != nil {
			s.log("Не удалось отправить письмо", "error", err)
		}
	}
	// MAX — как и письмо, дополнительный канал без повторов при сбое.
	if s.max != nil {
		if maxID, ok, err := s.maxLinks.MaxUserID(ctx, userID); err != nil {
			s.log("MAX: не удалось прочитать привязку", "error", err)
		} else if ok {
			text := n.Title
			if n.Body != "" {
				text += "\n" + n.Body
			}
			if err := s.max.Send(ctx, maxID, text); err != nil {
				s.log("MAX: уведомление не отправлено", "error", err)
			}
		}
	}
	return nil
}

// MaxStatus — включён ли канал и привязан ли аккаунт пользователя.
func (s *Service) MaxStatus(ctx context.Context, user string) (enabled, linked bool, err error) {
	if s.max == nil {
		return false, false, nil
	}
	_, linked, err = s.maxLinks.MaxUserID(ctx, user)
	return true, linked, err
}

// MaxLinkURL выдаёт одноразовую ссылку привязки (10 минут).
func (s *Service) MaxLinkURL(ctx context.Context, user string) (string, error) {
	if s.max == nil {
		return "", ErrMaxDisabled
	}
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := base64.RawURLEncoding.EncodeToString(b)
	if err := s.maxLinks.CreateLinkCode(ctx, user, code, 10*time.Minute); err != nil {
		return "", err
	}
	return s.max.LinkURL(code), nil
}

func (s *Service) MaxUnlink(ctx context.Context, user string) error {
	if s.max == nil {
		return ErrMaxDisabled
	}
	return s.maxLinks.UnlinkUser(ctx, user)
}
