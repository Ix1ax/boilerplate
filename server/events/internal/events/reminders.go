package events

import (
	"context"
	"errors"
	"strings"
	"time"

	"example.com/hackathon/server/events/internal/weather"
	"github.com/jackc/pgx/v5"
)

// Напоминания — за сутки (day) и за два часа (soon) до начала, плюс кнопка организатора «Напомнить»
// (manual, не чаще раза в час). Подтверждённый участник получает в одном сообщении всё, что нужно
// на месте: время, адрес, точку сбора, что взять, погоду для уличной встречи и имя проводника.

// weatherTimeout — прогноз не должен задерживать карточку и напоминание: нет ответа — нет погоды.
const weatherTimeout = 4 * time.Second

var ErrRemindedRecently = errors.New("Напоминание уже отправлено меньше часа назад")

// Forecaster — источник прогноза (weather.Client); nil — погода выключена.
type Forecaster interface {
	At(ctx context.Context, lat, lng float64, t time.Time) (weather.Forecast, error)
}

// ReminderTarget — поля события для текста напоминания.
type ReminderTarget struct {
	EventID, Title, Location, Requirements, PrivateInfo, Kind string
	StartsAt                                                  time.Time
	Lat, Lng                                                  *float64
	Outdoor                                                   bool
}

// EventWeather — ответ GET /events/{id}/weather. Reason объясняет человеку, почему прогноза нет.
type EventWeather struct {
	Available bool              `json:"available"`
	Reason    string            `json:"reason"`
	Forecast  *weather.Forecast `json:"forecast"`
}

const reminderSelect = `SELECT e.id,e.title,e.location,e.requirements,e.private_info,e.starts_at,e.lat,e.lng,e.outdoor`

func scanReminderTarget(row pgx.Row, t *ReminderTarget, extra ...any) error {
	return row.Scan(append([]any{&t.EventID, &t.Title, &t.Location, &t.Requirements, &t.PrivateInfo, &t.StartsAt, &t.Lat, &t.Lng, &t.Outdoor}, extra...)...)
}

// DueReminders — события, которым пора отправить напоминание. Событие, созданное позже чем
// за сутки (за два часа), такое напоминание не получает: люди только что записались.
func (r *Repository) DueReminders(ctx context.Context) ([]ReminderTarget, error) {
	rows, err := r.db.Query(ctx, reminderSelect+`,k.kind FROM events e
CROSS JOIN (VALUES ('day', interval '24 hours', interval '2 hours'), ('soon', interval '2 hours', interval '0')) k(kind, lead, till)
WHERE e.status='published' AND e.starts_at > now() + k.till AND e.starts_at <= now() + k.lead
  AND e.created_at < e.starts_at - k.lead
  AND NOT EXISTS (SELECT 1 FROM event_reminders er WHERE er.event_id=e.id AND er.kind=k.kind)
ORDER BY e.starts_at LIMIT 100`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (ReminderTarget, error) {
		var t ReminderTarget
		return t, scanReminderTarget(row, &t, &t.Kind)
	})
}

// ReminderTargetFor — для кнопки «Напомнить»: только организатор, только до начала.
func (r *Repository) ReminderTargetFor(ctx context.Context, a Actor, eventID string) (ReminderTarget, error) {
	t := ReminderTarget{Kind: "manual"}
	var organizerID, status string
	err := scanReminderTarget(r.db.QueryRow(ctx, reminderSelect+`,e.organizer_id,e.status FROM events e WHERE e.id=$1`, eventID), &t, &organizerID, &status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return t, ErrNotFound
	case err != nil:
		return t, err
	case organizerID != a.ID:
		return t, ErrNotOrganizer
	case status != "published":
		return t, ErrEventClosed
	case !t.StartsAt.After(time.Now()):
		return t, ErrEventPast
	}
	return t, nil
}

// SendReminder отмечает отправку и ставит уведомления в одной транзакции: повтор задания или вторая
// реплика ничего не продублируют. Возвращает число получателей.
func (r *Repository) SendReminder(ctx context.Context, t ReminderTarget, kind, weatherLine string) (int, error) {
	sent := 0
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var current ReminderTarget
		var status string
		var created time.Time
		err := scanReminderTarget(tx.QueryRow(ctx, reminderSelect+`,e.status,e.created_at FROM events e WHERE e.id=$1 FOR UPDATE`, t.EventID), &current, &status, &created)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != "published" || !current.StartsAt.After(time.Now()) {
			return nil
		}
		now := time.Now()
		switch kind {
		case "day":
			if !current.StartsAt.After(now.Add(2*time.Hour)) || current.StartsAt.After(now.Add(24*time.Hour)) || !created.Before(current.StartsAt.Add(-24*time.Hour)) {
				return nil
			}
		case "soon":
			if current.StartsAt.After(now.Add(2*time.Hour)) || !created.Before(current.StartsAt.Add(-2*time.Hour)) {
				return nil
			}
		case "manual":
		default:
			return FieldError{"kind", "Неизвестный вид напоминания"}
		}
		// Пока запрашивали прогноз, организатор мог перенести встречу. Старую погоду не прикладываем.
		if !current.StartsAt.Equal(t.StartsAt) || !sameFloat(current.Lat, t.Lat) || !sameFloat(current.Lng, t.Lng) || !current.Outdoor {
			weatherLine = ""
		}
		t = current
		query := "INSERT INTO event_reminders(event_id,kind) VALUES($1,$2) ON CONFLICT DO NOTHING"
		if kind == "manual" {
			query = `INSERT INTO event_reminders(event_id,kind) VALUES($1,$2) ON CONFLICT (event_id,kind)
DO UPDATE SET sent_at=now() WHERE event_reminders.sent_at < now() - interval '1 hour'`
		}
		tag, err := tx.Exec(ctx, query, t.EventID, kind)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			if kind == "manual" {
				return ErrRemindedRecently
			}
			return nil
		}
		rows, err := tx.Query(ctx, `SELECT p.user_id, COALESCE(bpr.display_name,''),
COALESCE((SELECT string_agg(COALESCE(npr.display_name,''), ', ' ORDER BY n.created_at) FROM participations n
  LEFT JOIN profiles npr ON npr.user_id=n.user_id WHERE n.buddy_id=p.id AND n.status='confirmed'), '')
FROM participations p
LEFT JOIN participations bp ON bp.id=p.buddy_id LEFT JOIN profiles bpr ON bpr.user_id=bp.user_id
WHERE p.event_id=$1 AND p.status='confirmed'`, t.EventID)
		if err != nil {
			return err
		}
		type recipient struct{ userID, buddy, newcomers string }
		list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (recipient, error) {
			var x recipient
			return x, row.Scan(&x.userID, &x.buddy, &x.newcomers)
		})
		if err != nil {
			return err
		}
		for _, x := range list {
			body := reminderBody(t, weatherLine, x.buddy, x.newcomers, time.Now())
			if err = notify(ctx, tx, x.userID, "Напоминание: «"+t.Title+"»", body, "/events/"+t.EventID); err != nil {
				return err
			}
		}
		sent = len(list)
		return nil
	})
	return sent, err
}

// reminderBody — текст для одного участника. Длинные поля обрезаются: уведомление — не карточка.
func reminderBody(t ReminderTarget, weatherLine, buddy, newcomers string, now time.Time) string {
	lines := []string{whenText(t.StartsAt, now) + " · " + t.Location + "."}
	if t.PrivateInfo != "" {
		lines = append(lines, "Точка сбора: "+truncate(t.PrivateInfo, 250))
	}
	if t.Requirements != "" {
		lines = append(lines, "Возьмите: "+truncate(t.Requirements, 250))
	}
	if weatherLine != "" {
		lines = append(lines, "Погода: "+weatherLine+".")
	}
	if buddy != "" {
		lines = append(lines, "Вас встретит "+buddy+".")
	}
	if newcomers != "" {
		lines = append(lines, "Вы встречаете: "+newcomers+".")
	}
	return strings.Join(lines, "\n")
}

// whenText — «Сегодня в 19:00», «Завтра в 10:00» или полная дата — по местному времени.
func whenText(t, now time.Time) string {
	t, now = t.In(localZone), now.In(localZone)
	hm := t.Format("15:04")
	y1, m1, d1 := t.Date()
	y0, m0, d0 := now.Date()
	today := time.Date(y0, m0, d0, 0, 0, 0, 0, localZone)
	switch time.Date(y1, m1, d1, 0, 0, 0, 0, localZone).Sub(today) {
	case 0:
		return "Сегодня в " + hm
	case 24 * time.Hour:
		return "Завтра в " + hm
	}
	return humanTime(t)
}

// SendReminders — фоновое задание (platform.Every): прогноз запрашивается вне транзакции.
func (s *Service) SendReminders(ctx context.Context) error {
	due, err := s.Store.DueReminders(ctx)
	if err != nil {
		return err
	}
	var failed error
	for _, t := range due {
		if _, err := s.Store.SendReminder(ctx, t, t.Kind, s.weatherLine(ctx, t)); err != nil {
			failed = errors.Join(failed, err)
		}
	}
	return failed
}

// Remind — кнопка организатора «Напомнить участникам сейчас».
func (s *Service) Remind(ctx context.Context, a Actor, eventID string) (int, error) {
	t, err := s.Store.ReminderTargetFor(ctx, a, eventID)
	if err != nil {
		return 0, err
	}
	return s.Store.SendReminder(ctx, t, "manual", s.weatherLine(ctx, t))
}

func (s *Service) weatherLine(ctx context.Context, t ReminderTarget) string {
	if !t.Outdoor || t.Lat == nil || s.Weather == nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, weatherTimeout)
	defer cancel()
	f, err := s.Weather.At(ctx, *t.Lat, *t.Lng, t.StartsAt)
	if err != nil {
		return ""
	}
	return f.Summary()
}

// EventWeather — прогноз для карточки. Причина отсутствия — человеческим языком, без ошибок API.
func (s *Service) EventWeather(ctx context.Context, a Actor, eventID string) (EventWeather, error) {
	e, err := s.Store.GetEvent(ctx, a, eventID)
	if err != nil {
		return EventWeather{}, err
	}
	no := func(reason string) (EventWeather, error) { return EventWeather{Reason: reason}, nil }
	switch {
	case !e.Outdoor:
		return no("Встреча в помещении — погода не важна")
	case e.Lat == nil:
		return no("Организатор не отметил место на карте")
	case s.Weather == nil:
		return no("Прогноз погоды отключён")
	case !e.StartsAt.After(time.Now()):
		return no("Встреча уже началась")
	}
	ctx, cancel := context.WithTimeout(ctx, weatherTimeout)
	defer cancel()
	f, err := s.Weather.At(ctx, *e.Lat, *e.Lng, e.StartsAt)
	switch {
	case errors.Is(err, weather.ErrTooFar):
		return no("Прогноз появится за две недели до встречи")
	case err != nil:
		return no("Прогноз сейчас недоступен")
	}
	return EventWeather{Available: true, Forecast: &f}, nil
}
