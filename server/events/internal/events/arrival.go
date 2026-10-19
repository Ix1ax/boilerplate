package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ArrivalInput struct {
	Status      string `json:"status"`
	ETAMinutes  int    `json:"eta_minutes"`
	Description string `json:"description"`
}

type Arrival struct {
	UserID      string     `json:"user_id"`
	Name        string     `json:"display_name"`
	Status      string     `json:"status"`
	ETAMinutes  int        `json:"eta_minutes"`
	Description string     `json:"description"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

func validateArrival(in *ArrivalInput) error {
	in.Description = strings.TrimSpace(in.Description)
	if in.Status != "arriving" && in.Status != "late" && in.Status != "here" {
		return FieldError{"status", "Выберите arriving, late или here"}
	}
	if in.ETAMinutes < 0 || in.ETAMinutes > 120 || (in.Status == "late" && in.ETAMinutes == 0) || (in.Status != "late" && in.ETAMinutes != 0) {
		return FieldError{"eta_minutes", "Для опоздания укажите от 1 до 120 минут, для остальных статусов — 0"}
	}
	if utf8.RuneCountInString(in.Description) > 200 {
		return FieldError{"description", "Описание — не длиннее 200 символов"}
	}
	return nil
}

func arrivalWindow(now, start, end time.Time) bool {
	return !now.Before(start.Add(-30*time.Minute)) && !now.After(end.Add(30*time.Minute))
}

func arrivalAccess(ctx context.Context, tx pgx.Tx, a Actor, eventID string) error {
	var start, end time.Time
	var status string
	var allowed bool
	err := tx.QueryRow(ctx, `SELECT e.starts_at,COALESCE(e.ends_at,e.starts_at+interval '2 hours'),e.status,
e.organizer_id=$2 OR EXISTS(SELECT 1 FROM participations p WHERE p.event_id=e.id AND p.user_id=$2 AND p.status='confirmed')
FROM events e WHERE e.id=$1 FOR SHARE`, eventID, a.ID).Scan(&start, &end, &status, &allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !allowed {
		return ErrNotFound
	}
	if status != "published" {
		return ErrEventClosed
	}
	if !arrivalWindow(time.Now(), start, end) {
		return fmt.Errorf("%w: отметки доступны за 30 минут до начала и до 30 минут после окончания", platform.ErrTransition)
	}
	return nil
}

const arrivalRoster = `SELECT p.user_id,COALESCE(pr.display_name,'Участник'),COALESCE(ar.status,'unknown'),COALESCE(ar.eta_minutes,0),COALESCE(ar.description,''),ar.updated_at
FROM (SELECT organizer_id user_id FROM events WHERE id=$1 UNION SELECT user_id FROM participations WHERE event_id=$1 AND status='confirmed')p
LEFT JOIN profiles pr ON pr.user_id=p.user_id LEFT JOIN event_arrivals ar ON ar.event_id=$1 AND ar.user_id=p.user_id`

func (r *Repository) Arrivals(ctx context.Context, a Actor, id string) ([]Arrival, error) {
	out := []Arrival{}
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		if err := arrivalAccess(ctx, tx, a, id); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, arrivalRoster+" ORDER BY p.user_id", id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p Arrival
			if err = rows.Scan(&p.UserID, &p.Name, &p.Status, &p.ETAMinutes, &p.Description, &p.UpdatedAt); err != nil {
				return err
			}
			out = append(out, p)
		}
		return rows.Err()
	})
	return out, err
}

func (r *Repository) SetArrival(ctx context.Context, a Actor, id string, in ArrivalInput) (Arrival, error) {
	if err := validateArrival(&in); err != nil {
		return Arrival{}, err
	}
	var out Arrival
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		// Сериализуем с исключением участника и отменой мероприятия: отменённая запись не оставит свежую отметку.
		if _, err := tx.Exec(ctx, "SELECT id FROM events WHERE id=$1 FOR UPDATE", id); err != nil {
			return err
		}
		if err := arrivalAccess(ctx, tx, a, id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO event_arrivals(event_id,user_id,status,eta_minutes,description)VALUES($1,$2,$3,$4,$5)
ON CONFLICT(event_id,user_id)DO UPDATE SET status=EXCLUDED.status,eta_minutes=EXCLUDED.eta_minutes,description=EXCLUDED.description,updated_at=now()`, id, a.ID, in.Status, in.ETAMinutes, in.Description)
		if err != nil {
			return err
		}
		err = tx.QueryRow(ctx, arrivalRoster+" WHERE p.user_id=$2", id, a.ID).Scan(&out.UserID, &out.Name, &out.Status, &out.ETAMinutes, &out.Description, &out.UpdatedAt)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT organizer_id FROM events WHERE id=$1 UNION SELECT user_id FROM participations WHERE event_id=$1 AND status='confirmed'`, id)
		if err != nil {
			return err
		}
		users := []string{}
		for rows.Next() {
			var user string
			if err = rows.Scan(&user); err != nil {
				rows.Close()
				return err
			}
			users = append(users, user)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return err
		}
		// Приватные отметки не публикуем в общедоступный room:event: канал содержит только сигнал для его участников.
		for _, user := range users {
			payload, _ := json.Marshal(map[string]any{"channel": "user:" + user, "from": "events", "data": map[string]string{"type": "arrival.changed", "event_id": id}, "at": time.Now().UTC()})
			if _, err = tx.Exec(ctx, `INSERT INTO outbox(id,type,user_id,request_id,payload)VALUES($1,'realtime.publish',$2,$3,$4)`, uuid.NewString(), user, platform.RequestID(ctx), payload); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}
