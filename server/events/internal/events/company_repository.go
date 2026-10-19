package events

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// companyMatch — условие «запрос r подходит событию e». Одно на оба направления подбора:
// при публикации события ищем запросы, при создании запроса — уже опубликованные события.
//   - направление совпадает, событие опубликовано и ещё не началось, организатор — не сам автор запроса;
//   - уровень совместим: «не важно» с любой стороны или точное совпадение;
//   - время суток и день недели — по местному времени события; пустой список — подходит любое;
//   - расстояние проверяем, только если точка есть и у запроса (с радиусом), и у события:
//     событие без координат не отсекаем — у него есть понятный адрес.
const companyMatch = `r.status='active' AND r.expires_at > now() AND r.sport_id=e.sport_id
AND e.status='published' AND e.starts_at > now() AND r.user_id <> e.organizer_id
AND (e.level='any' OR r.level='any' OR e.level=r.level)
AND (cardinality(r.day_parts)=0 OR (CASE
      WHEN extract(hour FROM e.starts_at AT TIME ZONE 'Asia/Krasnoyarsk') < 12 THEN 'morning'
      WHEN extract(hour FROM e.starts_at AT TIME ZONE 'Asia/Krasnoyarsk') < 17 THEN 'day'
      ELSE 'evening' END) = ANY(r.day_parts))
AND (cardinality(r.weekdays)=0 OR extract(isodow FROM e.starts_at AT TIME ZONE 'Asia/Krasnoyarsk')::smallint = ANY(r.weekdays))
AND (r.radius_km IS NULL OR e.lat IS NULL OR 2 * 6371 * asin(sqrt(
      power(sin(radians(e.lat - r.lat) / 2), 2) +
      cos(radians(r.lat)) * cos(radians(e.lat)) * power(sin(radians(e.lng - r.lng) / 2), 2))) <= r.radius_km)`

// $1 — текущий пользователь. Истёкший активный запрос показываем как expired, не меняя строку.
const companySelect = `SELECT r.id,s.id,s.name,s.is_custom,r.level,r.day_parts,r.weekdays::int[],r.area,r.lat,r.lng,r.radius_km,r.note,
CASE WHEN r.status='active' AND r.expires_at <= now() THEN 'expired' ELSE r.status END,r.expires_at,r.created_at,
(SELECT count(*) FROM company_matches m JOIN events e ON e.id=m.event_id
  WHERE m.request_id=r.id AND e.status='published' AND COALESCE(e.ends_at, e.starts_at) > now()),
(SELECT count(*) FROM company_requests o WHERE o.sport_id=r.sport_id AND o.user_id <> r.user_id
  AND o.status='active' AND o.expires_at > now())
FROM company_requests r JOIN sports s ON s.id=r.sport_id
WHERE r.user_id=$1`

func scanCompanyRequest(row pgx.Row) (CompanyRequest, error) {
	var c CompanyRequest
	err := row.Scan(&c.ID, &c.Sport.ID, &c.Sport.Name, &c.Sport.IsCustom, &c.Level, &c.DayParts, &c.Weekdays, &c.Area,
		&c.Lat, &c.Lng, &c.RadiusKm, &c.Note, &c.Status, &c.ExpiresAt, &c.CreatedAt, &c.MatchesCount, &c.OthersCount)
	return c, err
}

func (r *Repository) MyCompanyRequest(ctx context.Context, a Actor, id string) (CompanyRequest, error) {
	c, err := scanCompanyRequest(r.db.QueryRow(ctx, companySelect+" AND r.id=$2", a.ID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return CompanyRequest{}, ErrNotFound
	}
	return c, err
}

func (r *Repository) MyCompanyRequests(ctx context.Context, a Actor) ([]CompanyRequest, error) {
	rows, err := r.db.Query(ctx, companySelect+" ORDER BY (r.status='active' AND r.expires_at > now()) DESC, r.created_at DESC LIMIT 50", a.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CompanyRequest{}
	for rows.Next() {
		c, err := scanCompanyRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateCompanyRequest сохраняет запрос и сразу привязывает уже опубликованные подходящие события:
// человек видит их списком, поэтому уведомления о них не нужны.
func (r *Repository) CreateCompanyRequest(ctx context.Context, a Actor, in CompanyRequestInput) (CompanyRequest, error) {
	id := uuid.NewString()
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		sportID, err := resolveSport(ctx, tx, a, in.SportID, in.SportName)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO company_requests(id,user_id,sport_id,level,day_parts,weekdays,area,lat,lng,radius_km,note,expires_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (user_id, sport_id) WHERE status='active' DO NOTHING`,
			id, a.ID, sportID, in.Level, in.DayParts, in.Weekdays, in.Area, in.Lat, in.Lng, in.RadiusKm, in.Note, time.Now().Add(companyRequestTTL))
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrRequestExists
		}
		_, err = tx.Exec(ctx, `INSERT INTO company_matches(request_id,event_id)
SELECT r.id, e.id FROM company_requests r JOIN events e ON `+companyMatch+` WHERE r.id=$1
ON CONFLICT DO NOTHING`, id)
		return err
	})
	if err != nil {
		return CompanyRequest{}, err
	}
	return r.MyCompanyRequest(ctx, a, id)
}

// CloseCompanyRequest — «компания нашлась» или «больше не ищу»: уведомления прекращаются.
func (r *Repository) CloseCompanyRequest(ctx context.Context, a Actor, id string) error {
	tag, err := r.db.Exec(ctx, "UPDATE company_requests SET status='closed', updated_at=now() WHERE id=$1 AND user_id=$2 AND status='active'", id, a.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// matchNewEvent вызывается в транзакции публикации: находит подходящие запросы, записывает совпадения
// и ставит в outbox уведомления. Если транзакция откатится, не уйдёт ни событие, ни уведомления.
func matchNewEvent(ctx context.Context, tx pgx.Tx, eventID string) error {
	rows, err := tx.Query(ctx, `WITH m AS (
  INSERT INTO company_matches(request_id,event_id,notified)
  SELECT r.id, e.id, true FROM events e JOIN company_requests r ON `+companyMatch+` WHERE e.id=$1
  ON CONFLICT DO NOTHING RETURNING request_id)
SELECT r.user_id, s.name, e.title, e.starts_at, e.location
FROM m JOIN company_requests r ON r.id=m.request_id JOIN events e ON e.id=$1 JOIN sports s ON s.id=e.sport_id`, eventID)
	if err != nil {
		return err
	}
	type invite struct {
		userID, sport, title, location string
		startsAt                       time.Time
	}
	var invites []invite
	for rows.Next() {
		var i invite
		if err = rows.Scan(&i.userID, &i.sport, &i.title, &i.startsAt, &i.location); err != nil {
			rows.Close()
			return err
		}
		invites = append(invites, i)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	// Запросы читаем целиком до отправки: в pgx нельзя выполнять новые запросы, пока открыт rows.
	for _, i := range invites {
		body := fmt.Sprintf("«%s», %s — %s. Вы искали компанию на «%s»: событие подходит по уровню и времени.",
			i.title, humanTime(i.startsAt), i.location, i.sport)
		if err := notify(ctx, tx, i.userID, "Нашлась компания: "+i.sport, body, "/events/"+eventID); err != nil {
			return err
		}
	}
	return nil
}

// Demand считает чужие активные запросы на направление с совместимым уровнем. Своё направление,
// которого ещё нет в справочнике, — это ноль, а не ошибка: спроса пока просто нет.
func (r *Repository) Demand(ctx context.Context, a Actor, sportID, sportName, level string) (Demand, error) {
	d := Demand{ByDayPart: map[string]int{"morning": 0, "day": 0, "evening": 0}, ByWeekday: make([]int, 7)}
	var id string
	err := r.db.QueryRow(ctx, "SELECT id FROM sports WHERE NOT hidden AND (id::text=$1 OR ($1='' AND lower(btrim(name))=lower($2)))", sportID, sportName).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, nil
	}
	if err != nil {
		return d, err
	}
	d.SportID = &id
	const cond = `FROM company_requests r WHERE r.sport_id=$1 AND r.status='active' AND r.expires_at > now()
AND r.user_id <> $2 AND (r.level='any' OR $3='any' OR r.level=$3)`
	// Пустой список времени означает «когда угодно» — такой человек учитывается в каждой части дня.
	var morning, day, evening int
	err = r.db.QueryRow(ctx, `SELECT count(*),
count(*) FILTER (WHERE cardinality(r.day_parts)=0 OR 'morning'=ANY(r.day_parts)),
count(*) FILTER (WHERE cardinality(r.day_parts)=0 OR 'day'=ANY(r.day_parts)),
count(*) FILTER (WHERE cardinality(r.day_parts)=0 OR 'evening'=ANY(r.day_parts)) `+cond, id, a.ID, level).
		Scan(&d.Count, &morning, &day, &evening)
	if err != nil {
		return d, err
	}
	d.ByDayPart = map[string]int{"morning": morning, "day": day, "evening": evening}
	rows, err := r.db.Query(ctx, `SELECT w, count(*) FROM generate_series(1,7) w, LATERAL (SELECT 1 `+cond+`
AND (cardinality(r.weekdays)=0 OR w=ANY(r.weekdays))) x GROUP BY w`, id, a.ID, level)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var w, n int
		if err = rows.Scan(&w, &n); err != nil {
			return d, err
		}
		d.ByWeekday[w-1] = n
	}
	return d, rows.Err()
}

// Красноярск живёт в UTC+7 круглый год, поэтому фиксированного пояса достаточно и образу не нужна tzdata.
var localZone = time.FixedZone("Asia/Krasnoyarsk", 7*60*60)

var (
	weekdayNames = [...]string{"вс", "пн", "вт", "ср", "чт", "пт", "сб"}
	monthNames   = [...]string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}
)

// humanTime — «сб, 24 октября в 10:00» по местному времени: текст уведомления читают люди, а не клиент.
func humanTime(t time.Time) string {
	t = t.In(localZone)
	return fmt.Sprintf("%s, %d %s в %02d:%02d", weekdayNames[t.Weekday()], t.Day(), monthNames[t.Month()-1], t.Hour(), t.Minute())
}
