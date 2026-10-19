package events

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/jackc/pgx/v5"
)

// Правка события организатором. Режим записи, направление и «от чьего имени» после публикации
// не меняются: записавшиеся соглашались именно на них. Перенос времени или места — с уведомлением
// всем записавшимся «было → стало»; правка условий — с уведомлением «проверьте, что изменилось».

var ErrNotEventOwner = errors.New("Менять мероприятие может только его организатор")

// EventPatch — изменяемые поля; nil — не трогаем. Координаты задаются парой.
type EventPatch struct {
	Title        *string    `json:"title"`
	Description  *string    `json:"description"`
	StartsAt     *time.Time `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at"`
	Location     *string    `json:"location"`
	Lat          *float64   `json:"lat"`
	Lng          *float64   `json:"lng"`
	Level        *string    `json:"level"`
	Requirements *string    `json:"requirements"`
	PriceNote    *string    `json:"price_note"`
	PrivateInfo  *string    `json:"private_info"`
	Capacity     *int       `json:"capacity"`
	Outdoor      *bool      `json:"outdoor"`
}

// EventChanges — что именно поменялось: от этого зависят уведомления и сброс напоминаний.
type EventChanges struct {
	Time, Place, Conditions bool
	OldStartsAt             time.Time
	OldLocation             string
	Expected                *EventInput
}

func (c EventChanges) any() bool { return c.Time || c.Place || c.Conditions }

// UpdateEvent сливает правку с текущим состоянием и проверяет результат теми же правилами, что и создание.
func (s *Service) UpdateEvent(ctx context.Context, a Actor, id string, p EventPatch) (Event, error) {
	cur, err := s.Store.GetEvent(ctx, a, id)
	switch {
	case err != nil:
		return Event{}, err
	case !cur.IsOrganizer:
		return Event{}, ErrNotEventOwner
	case cur.Status != "published":
		return Event{}, ErrEventClosed
	case !cur.StartsAt.After(time.Now()):
		return Event{}, ErrEventPast
	}
	in := EventInput{Format: cur.Format, Title: cur.Title, Description: cur.Description, SportID: cur.Sport.ID, StartsAt: cur.StartsAt, EndsAt: cur.EndsAt,
		Location: cur.Location, Lat: cur.Lat, Lng: cur.Lng, Level: cur.Level, Requirements: cur.Requirements, PriceNote: cur.PriceNote,
		PrivateInfo: cur.PrivateInfo, JoinMode: cur.JoinMode, Capacity: cur.Capacity, MinParticipants: cur.MinParticipants,
		QuorumDeadline: cur.QuorumDeadline, Outdoor: cur.Outdoor}
	expected := in
	set(&in.Title, p.Title)
	set(&in.Description, p.Description)
	set(&in.StartsAt, p.StartsAt)
	set(&in.Location, p.Location)
	set(&in.Level, p.Level)
	set(&in.Requirements, p.Requirements)
	set(&in.PriceNote, p.PriceNote)
	set(&in.PrivateInfo, p.PrivateInfo)
	set(&in.Outdoor, p.Outdoor)
	if p.EndsAt != nil {
		in.EndsAt = p.EndsAt
	}
	if p.Capacity != nil {
		in.Capacity = p.Capacity
	}
	if p.Lat != nil || p.Lng != nil {
		in.Lat, in.Lng = p.Lat, p.Lng
	}
	if p.StartsAt != nil && in.EndsAt != nil && cur.EndsAt != nil && p.EndsAt == nil {
		// Перенесли начало без конца — сохраняем длительность, а не ломаем «конец раньше начала».
		end := in.StartsAt.Add(cur.EndsAt.Sub(cur.StartsAt))
		in.EndsAt = &end
	}
	if p.StartsAt != nil && in.MinParticipants != nil {
		in.QuorumDeadline = nil // срок сбора состава считается заново от нового начала
	}
	// Неизменённый срок уже мог пройти: это не мешает исправить место встречи.
	if err := validateEvent(&in, time.Now(), p.StartsAt != nil); err != nil {
		return Event{}, err
	}
	changes := EventChanges{Expected: &expected, OldStartsAt: cur.StartsAt, OldLocation: cur.Location,
		Time:  !in.StartsAt.Equal(cur.StartsAt) || !sameTime(in.EndsAt, cur.EndsAt),
		Place: in.Location != cur.Location || !sameFloat(in.Lat, cur.Lat) || !sameFloat(in.Lng, cur.Lng),
		Conditions: in.Requirements != cur.Requirements || in.PrivateInfo != cur.PrivateInfo || in.PriceNote != cur.PriceNote ||
			in.Level != cur.Level}
	return s.Store.UpdateEvent(ctx, a, id, in, changes)
}

func set[T any](dst *T, v *T) {
	if v != nil {
		*dst = *v
	}
}

func sameTime(a, b *time.Time) bool {
	return a == nil && b == nil || a != nil && b != nil && a.Equal(*b)
}

func sameFloat(a, b *float64) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

// UpdateEvent сохраняет правку под блокировкой события: права и «мест не меньше записавшихся»
// проверяются ещё раз, потому что между чтением и записью могли записаться люди.
func (r *Repository) UpdateEvent(ctx context.Context, a Actor, id string, in EventInput, c EventChanges) (Event, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		e, err := lockEvent(ctx, tx, id)
		switch {
		case err != nil:
			return err
		case e.OrganizerID != a.ID:
			return ErrNotEventOwner
		case e.Status != "published":
			return ErrEventClosed
		}
		if !e.StartsAt.After(time.Now()) {
			return ErrEventPast
		}
		if c.Expected != nil {
			current, err := getEvent(ctx, tx, a, id)
			if err != nil {
				return err
			}
			snapshot := EventInput{Format: current.Format, Title: current.Title, Description: current.Description, SportID: current.Sport.ID, StartsAt: current.StartsAt, EndsAt: current.EndsAt,
				Location: current.Location, Lat: current.Lat, Lng: current.Lng, Level: current.Level, Requirements: current.Requirements, PriceNote: current.PriceNote,
				PrivateInfo: current.PrivateInfo, JoinMode: current.JoinMode, Capacity: current.Capacity, MinParticipants: current.MinParticipants,
				QuorumDeadline: current.QuorumDeadline, Outdoor: current.Outdoor}
			// Полный снимок сравниваем под блокировкой: устаревшая правка не затрёт изменения другого запроса.
			// Времена PostgreSQL нормализуются в UTC, поэтому сравниваем их отдельно через Equal.
			expected := *c.Expected
			sameDates := snapshot.StartsAt.Equal(expected.StartsAt) && sameTime(snapshot.EndsAt, expected.EndsAt) && sameTime(snapshot.QuorumDeadline, expected.QuorumDeadline)
			snapshot.StartsAt = expected.StartsAt
			snapshot.EndsAt = expected.EndsAt
			snapshot.QuorumDeadline = expected.QuorumDeadline
			if !sameDates || !reflect.DeepEqual(snapshot, expected) {
				return fmt.Errorf("%w: мероприятие уже изменилось, обновите страницу и повторите правку", platform.ErrTransition)
			}
		}
		if in.Capacity != nil {
			n, err := confirmedCount(ctx, tx, id)
			if err != nil {
				return err
			}
			if n > *in.Capacity {
				return FieldError{"capacity", fmt.Sprintf("Уже записано %d — мест не может быть меньше", n)}
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE events SET title=$2, description=$3, starts_at=$4, ends_at=$5, location=$6, lat=$7, lng=$8,
level=$9, requirements=$10, price_note=$11, private_info=$12, capacity=$13, quorum_deadline=$14, outdoor=$15, updated_at=now() WHERE id=$1`,
			id, in.Title, in.Description, in.StartsAt, in.EndsAt, in.Location, in.Lat, in.Lng, in.Level, in.Requirements,
			in.PriceNote, in.PrivateInfo, in.Capacity, in.QuorumDeadline, in.Outdoor); err != nil {
			return err
		}
		// Добавленные места сначала получает лист ожидания, иначе их займёт следующий записавшийся.
		if e, err = lockEvent(ctx, tx, id); err != nil {
			return err
		}
		if err = promoteWaitlist(ctx, tx, id, e); err != nil {
			return err
		}
		if c.Time {
			// Напоминания «за сутки» и «за два часа» должны прийти заново — уже от нового времени.
			if _, err = tx.Exec(ctx, "DELETE FROM event_reminders WHERE event_id=$1 AND kind IN ('day','soon')", id); err != nil {
				return err
			}
		}
		if err = record(ctx, tx, "event", id, platform.Transition{From: "published", To: "published", Actor: actorLabel(a), Comment: c.summary()}); err != nil {
			return err
		}
		if !c.any() {
			return nil
		}
		title, body := "Организатор обновил условия: «"+in.Title+"»", "Проверьте, что взять и где встретиться — подробности на странице мероприятия."
		if c.Time || c.Place {
			title = "Перенос: «" + in.Title + "»"
			body = fmt.Sprintf("Было: %s · %s.\nСтало: %s · %s.", humanTime(c.OldStartsAt), c.OldLocation, humanTime(in.StartsAt), in.Location)
		}
		rows, err := tx.Query(ctx, "SELECT user_id FROM participations WHERE event_id=$1 AND status IN ('confirmed','pending','waitlist')", id)
		if err != nil {
			return err
		}
		users, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, u := range users {
			if err = notify(ctx, tx, u, title, body, "/events/"+id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Event{}, err
	}
	return r.GetEvent(ctx, a, id)
}

func (c EventChanges) summary() string {
	var parts []string
	if c.Time {
		parts = append(parts, "время")
	}
	if c.Place {
		parts = append(parts, "место")
	}
	if c.Conditions {
		parts = append(parts, "условия")
	}
	if len(parts) == 0 {
		return "Изменено описание"
	}
	return "Изменено: " + strings.Join(parts, ", ")
}
