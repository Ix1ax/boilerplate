package events

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var ErrSocialConflict = errors.New("Действие недоступно в текущем состоянии")

// SocialStore отделён от основного Store: старые обработчики и их тестовые хранилища
// не должны зависеть от личной переписки.
type SocialStore interface {
	Friendship(context.Context, Actor, string, string) (any, error)
	Friends(context.Context, Actor, SocialPage) (SocialResult, error)
	OpenDirectChat(context.Context, Actor, string) (any, error)
	Chats(context.Context, Actor, SocialPage) (SocialResult, error)
	DirectMessages(context.Context, Actor, string, SocialPage) (SocialResult, error)
	SendDirectMessage(context.Context, Actor, string, string) (any, error)
	CreateReview(context.Context, Actor, string, int, string) (any, error)
	ProfileReviews(context.Context, Actor, string, SocialPage) (SocialResult, error)
	HideReview(context.Context, Actor, string, string) error
	CreateStory(context.Context, Actor, string) (any, error)
	Stories(context.Context, Actor, SocialPage) (SocialResult, error)
	DeleteStory(context.Context, Actor, string, string) error
	ReserveStoryMedia(context.Context, Actor, string) (string, error)
	AttachStoryMedia(context.Context, Actor, string, string, string, int64) error
	StoryMediaAccess(context.Context, Actor, string) (StoryMediaInfo, error)
	ClubMembership(context.Context, Actor, string, bool) error
	ClubMembers(context.Context, Actor, string, SocialPage) (SocialResult, error)
}

type SocialPage struct {
	Before time.Time
	ID     string
	Limit  int
}
type SocialResult struct {
	Items      []json.RawMessage `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}
type socialCursor struct {
	At time.Time `json:"at"`
	ID string    `json:"id"`
}

func parseSocialPage(cursor string, limit int) (SocialPage, error) {
	p := SocialPage{Before: time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC), ID: "ffffffff-ffff-ffff-ffff-ffffffffffff", Limit: limit}
	if limit < 1 || limit > 100 {
		return p, FieldError{"limit", "Размер страницы — от 1 до 100"}
	}
	if cursor != "" {
		if len(cursor) > 512 {
			return p, FieldError{"cursor", "Некорректный курсор"}
		}
		b, err := base64.RawURLEncoding.DecodeString(cursor)
		var c socialCursor
		if err != nil || json.Unmarshal(b, &c) != nil || c.At.IsZero() || c.At.Year() < 1 || c.At.Year() > 9999 || c.ID == "" || len(c.ID) > 200 {
			return p, FieldError{"cursor", "Некорректный курсор"}
		}
		p.Before, p.ID = c.At, c.ID
	}
	return p, nil
}
func socialBody(v string, max int) (string, error) {
	v = strings.TrimSpace(v)
	if !utf8.ValidString(v) || utf8.RuneCountInString(v) < 1 || utf8.RuneCountInString(v) > max {
		return "", FieldError{"body", "Проверьте длину текста"}
	}
	return v, nil
}
func socialPeer(a Actor, peer string) error {
	if peer == a.ID || strings.TrimSpace(peer) == "" || len(peer) > 200 {
		return FieldError{"user_id", "Выберите другого пользователя"}
	}
	return nil
}
func socialUUID(id string) error {
	if uuid.Validate(id) != nil {
		return ErrNotFound
	}
	return nil
}
