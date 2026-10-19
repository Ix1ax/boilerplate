package events

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/jackc/pgx/v5"
)

var (
	ErrVideoUnavailable = errors.New("Видеовстречи пока не настроены")
	ErrVideoWindow      = errors.New("Войти можно за 15 минут до начала и до окончания встречи")
	ErrVideoOffline     = errors.New("Это мероприятие проходит очно")
)

type EventFormat struct {
	EventID string `json:"event_id"`
	Format  string `json:"format"`
}

type VideoAccess struct {
	EventFormat
	OrganizerID string
	Status      string
	StartsAt    time.Time
	EndsAt      *time.Time
	Confirmed   bool
}

type OnlineStore interface {
	GetEventFormat(context.Context, Actor, string) (EventFormat, error)
	SetEventFormat(context.Context, Actor, string, string) (EventFormat, error)
	GetVideoAccess(context.Context, Actor, string) (VideoAccess, error)
}

type VideoConfig struct{ URL, APIKey, APISecret string }

func (c VideoConfig) Valid() bool {
	u, err := url.Parse(c.URL)
	return err == nil && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == "" &&
		(u.Scheme == "wss" || u.Scheme == "ws" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")) &&
		strings.TrimSpace(c.APIKey) != "" && len(c.APISecret) >= 32
}

type VideoToken struct {
	URL       string    `json:"url"`
	Room      string    `json:"room"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Короткий билет разрешает вход только в комнату конкретного события. Прав на создание
// чужих комнат, запись звонка и администрирование участников в нём нет.
func IssueVideoToken(c VideoConfig, a Actor, v VideoAccess, now time.Time) (VideoToken, error) {
	if a.ID == "" || a.ID != v.OrganizerID && !v.Confirmed {
		return VideoToken{}, ErrNotParticipant
	}
	if v.Status != "published" {
		return VideoToken{}, ErrEventClosed
	}
	if v.Format != "online" {
		return VideoToken{}, ErrVideoOffline
	}
	end := v.StartsAt.Add(90 * time.Minute)
	if v.EndsAt != nil {
		end = *v.EndsAt
	}
	if now.Before(v.StartsAt.Add(-15*time.Minute)) || !now.Before(end) {
		return VideoToken{}, ErrVideoWindow
	}
	if !c.Valid() {
		return VideoToken{}, ErrVideoUnavailable
	}
	expiry := now.Add(2 * time.Minute)
	if expiry.After(end) {
		expiry = end
	}
	room := "event_" + v.EventID
	claims := map[string]any{
		"iss": c.APIKey, "sub": a.ID, "name": displayName(a), "iat": now.Unix(), "nbf": now.Unix(), "exp": expiry.Unix(),
		"video": map[string]any{"roomJoin": true, "room": room, "canPublish": true, "canSubscribe": true, "canPublishData": false, "canUpdateOwnMetadata": false},
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return VideoToken{}, err
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(c.APISecret))
	_, _ = mac.Write([]byte(body))
	return VideoToken{c.URL, room, body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), expiry}, nil
}

func (r *Repository) GetEventFormat(ctx context.Context, a Actor, id string) (EventFormat, error) {
	var out EventFormat
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := getEvent(ctx, tx, a, id); err != nil {
			return err
		}
		return tx.QueryRow(ctx, "SELECT id,format FROM events WHERE id=$1", id).Scan(&out.EventID, &out.Format)
	})
	return out, err
}

func (r *Repository) SetEventFormat(ctx context.Context, a Actor, id, format string) (EventFormat, error) {
	if format != "offline" && format != "online" {
		return EventFormat{}, FieldError{"format", "Формат: offline или online"}
	}
	out := EventFormat{EventID: id, Format: format}
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		// Та же блокировка, что у отмены и изменения события: формат не меняется посреди звонка.
		var owner, status, old string
		var starts time.Time
		if err := tx.QueryRow(ctx, "SELECT organizer_id,status,starts_at,format FROM events WHERE id=$1 FOR UPDATE", id).Scan(&owner, &status, &starts, &old); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if owner != a.ID {
			return ErrNotEventOwner
		}
		if status != "published" {
			return ErrEventClosed
		}
		// После открытия входа участники уже могли подключиться к комнате.
		if !time.Now().Before(starts.Add(-15 * time.Minute)) {
			return ErrEventPast
		}
		if old == format {
			return nil
		}
		if _, err := tx.Exec(ctx, "UPDATE events SET format=$2,updated_at=now() WHERE id=$1 AND organizer_id=$3", id, format, a.ID); err != nil {
			return err
		}
		if err := record(ctx, tx, "event", id, platform.Transition{From: status, To: status, Actor: actorLabel(a), Comment: "Изменён формат встречи: " + format}); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, "SELECT user_id FROM participations WHERE event_id=$1 AND status IN ('confirmed','pending','waitlist')", id)
		if err != nil {
			return err
		}
		users, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		label := "очно"
		if format == "online" {
			label = "онлайн"
		}
		for _, u := range users {
			if err := notify(ctx, tx, u, "Изменился формат встречи", "Теперь встреча проходит "+label+". Проверьте условия участия.", "/events/"+id); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

func (r *Repository) GetVideoAccess(ctx context.Context, a Actor, id string) (VideoAccess, error) {
	var v VideoAccess
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT e.id,e.format,e.organizer_id,e.status,e.starts_at,e.ends_at,
 EXISTS(SELECT 1 FROM participations p WHERE p.event_id=e.id AND p.user_id=$2 AND p.status='confirmed')
 FROM events e WHERE e.id=$1 AND (e.organizer_id=$2 OR EXISTS(SELECT 1 FROM participations p WHERE p.event_id=e.id AND p.user_id=$2 AND p.status='confirmed')) FOR SHARE OF e`, id, a.ID).Scan(&v.EventID, &v.Format, &v.OrganizerID, &v.Status, &v.StartsAt, &v.EndsAt, &v.Confirmed)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return v, err
}
