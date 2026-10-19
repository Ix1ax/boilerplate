package events

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// Профиль и доверие. Имя приходит из Keycloak и здесь не меняется; человек правит только то,
// что помогает другим понять, подходит ли ему компания: о себе, город, уровень.

// OrganizerStats — «можно ли доверять организатору»: только факты из данных платформы, без оценок.
type OrganizerStats struct {
	// EventsHeld — прошедшие опубликованные встречи; PeopleCame — подтверждённые записи на них
	// (посещение платформа не проверяет, поэтому это «записались», а не «пришли»).
	EventsHeld     int       `json:"events_held"`
	PeopleCame     int       `json:"people_came"`
	UpcomingEvents int       `json:"upcoming_events"`
	Since          time.Time `json:"since"`
}

// ProfilePatch — что человек может поменять в своём профиле. nil — поле не трогаем.
type ProfilePatch struct {
	About *string `json:"about"`
	City  *string `json:"city"`
	// Level: beginner, amateur, pro или пустая строка — «не указывать».
	Level *string `json:"level"`
}

var profileLevels = map[string]bool{"": true, "beginner": true, "amateur": true, "pro": true}

func ValidateProfile(p *ProfilePatch) error {
	for _, f := range []**string{&p.About, &p.City, &p.Level} {
		if *f != nil {
			v := strings.TrimSpace(**f)
			*f = &v
		}
	}
	switch {
	case p.About != nil && utf8.RuneCountInString(*p.About) > 1000:
		return FieldError{"about", "О себе — не длиннее 1000 символов"}
	case p.City != nil && utf8.RuneCountInString(*p.City) > 100:
		return FieldError{"city", "Город — не длиннее 100 символов"}
	case p.Level != nil && !profileLevels[*p.Level]:
		return FieldError{"level", "Уровень: beginner, amateur, pro или пусто"}
	}
	return nil
}

func (s *Service) UpdateMyProfile(ctx context.Context, a Actor, p ProfilePatch) (Profile, error) {
	if err := ValidateProfile(&p); err != nil {
		return Profile{}, err
	}
	return s.Store.UpdateProfile(ctx, a, p)
}

// UpdateProfile меняет только переданные поля; пустой уровень хранится как NULL («не указан»).
func (r *Repository) UpdateProfile(ctx context.Context, a Actor, p ProfilePatch) (Profile, error) {
	_, err := r.db.Exec(ctx, `UPDATE profiles SET
about = COALESCE($2, about), city = COALESCE($3, city),
level = CASE WHEN $4::text IS NULL THEN level ELSE NULLIF($4, '') END
WHERE user_id=$1`, a.ID, p.About, p.City, p.Level)
	if err != nil {
		return Profile{}, err
	}
	return r.GetProfile(ctx, a, a.ID)
}

// organizerStats считает статистику по событиям человека (column=organizer_id, без организации)
// или организации (column=organization_id). since — когда появился на платформе.
func organizerStats(ctx context.Context, q querier, column, id string, since time.Time) (OrganizerStats, error) {
	st := OrganizerStats{Since: since}
	cond := "e." + column + "=$1"
	if column == "organizer_id" {
		cond += " AND e.organization_id IS NULL" // события от имени организации — её заслуга
	}
	err := q.QueryRow(ctx, `SELECT
count(*) FILTER (WHERE e.status='published' AND COALESCE(e.ends_at, e.starts_at) <= now()),
COALESCE(sum((SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='confirmed'))
  FILTER (WHERE e.status='published' AND COALESCE(e.ends_at, e.starts_at) <= now()), 0),
count(*) FILTER (WHERE e.status='published' AND e.starts_at > now())
FROM events e WHERE `+cond, id).Scan(&st.EventsHeld, &st.PeopleCame, &st.UpcomingEvents)
	return st, err
}

func (r *Repository) profileStats(ctx context.Context, id string) (*OrganizerStats, error) {
	var since time.Time
	if err := r.db.QueryRow(ctx, "SELECT created_at FROM profiles WHERE user_id=$1", id).Scan(&since); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	st, err := organizerStats(ctx, r.db, "organizer_id", id, since)
	return &st, err
}

func (r *Repository) OrganizationStats(ctx context.Context, id string) (OrganizerStats, error) {
	var since time.Time
	if err := r.db.QueryRow(ctx, "SELECT COALESCE(reviewed_at, created_at) FROM organizations WHERE id=$1", id).Scan(&since); err != nil {
		if err == pgx.ErrNoRows {
			return OrganizerStats{}, ErrNotFound
		}
		return OrganizerStats{}, err
	}
	return organizerStats(ctx, r.db, "organization_id", id, since)
}
