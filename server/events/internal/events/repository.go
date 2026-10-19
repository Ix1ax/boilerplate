package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository — Store на Postgres. Изменения, зависящие от состояния (запись, решение по заявке,
// модерация), блокируют строку события FOR UPDATE: два одновременных запроса не займут
// последнее место дважды и не обойдут снятие с публикации.
type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// EnsureProfile держит копию имени из Keycloak: авторы и участники показываются по имени,
// хотя пользователи живут в другой системе. Запись меняется, только если имя изменилось.
func (r *Repository) EnsureProfile(ctx context.Context, a Actor) error {
	_, err := r.db.Exec(ctx, `INSERT INTO profiles(user_id,display_name) VALUES($1,$2)
ON CONFLICT (user_id) DO UPDATE SET display_name=EXCLUDED.display_name
WHERE profiles.display_name IS DISTINCT FROM EXCLUDED.display_name`, a.ID, displayName(a))
	return err
}

func displayName(a Actor) string {
	name := strings.TrimSpace(a.Name)
	if name == "" {
		name, _, _ = strings.Cut(a.Email, "@")
	}
	if name == "" {
		name = "Участник"
	}
	if len([]rune(name)) > 100 {
		name = string([]rune(name)[:100])
	}
	return name
}

func actorLabel(a Actor) string {
	if a.Email != "" {
		return a.Email
	}
	return a.ID
}

func (r *Repository) Sports(ctx context.Context) ([]Sport, error) {
	rows, err := r.db.Query(ctx, "SELECT id,name,is_custom FROM sports WHERE NOT hidden ORDER BY is_custom, lower(name)")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Sport, error) {
		var s Sport
		return s, row.Scan(&s.ID, &s.Name, &s.IsCustom)
	})
}

// ── События ─────────────────────────────────────────────────────────────────

// $1 в запросах событий — всегда текущий пользователь: для его записи и признака «организатор».
const eventSelect = `SELECT e.id,e.title,e.description,s.id,s.name,s.is_custom,e.starts_at,e.ends_at,e.location,e.lat,e.lng,
e.level,e.requirements,e.price_note,e.private_info,e.join_mode,e.capacity,
(SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='confirmed'),
(SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='pending'),
(SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='waitlist'),
(SELECT count(*) FROM company_matches cm WHERE cm.event_id=e.id),
e.status,e.moderation_reason,e.organizer_id,COALESCE(pr.display_name,''),o.id,COALESCE(o.name,''),
mp.id,mp.status,mp.decision_reason,e.created_at,
e.min_participants,e.quorum_deadline,e.quorum_status,e.outdoor,
(SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='confirmed' AND p.first_time),
(SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='confirmed' AND p.buddy),
COALESCE(mp.first_time,false),COALESCE(mp.buddy,false),bp.user_id,COALESCE(bpr.display_name,'')
FROM events e
JOIN sports s ON s.id=e.sport_id
LEFT JOIN organizations o ON o.id=e.organization_id
LEFT JOIN profiles pr ON pr.user_id=e.organizer_id
LEFT JOIN participations mp ON mp.event_id=e.id AND mp.user_id=$1
LEFT JOIN participations bp ON bp.id=mp.buddy_id
LEFT JOIN profiles bpr ON bpr.user_id=bp.user_id`

// visibleEvent: опубликованное видят все; снятое, удалённое и отменённое — организатор,
// записавшиеся (с причиной) и админ ($2). Остальным оно «не существует» (404).
const visibleEvent = ` AND (e.status='published' OR e.organizer_id=$1 OR mp.id IS NOT NULL OR $2::bool)`

func scanEvent(row pgx.Row, a Actor) (Event, error) {
	var e Event
	var organizerID, organizerName, orgName string
	var orgID, mpID, mpStatus, mpReason, buddyID *string
	var mpFirstTime, mpBuddy bool
	var buddyName string
	err := row.Scan(&e.ID, &e.Title, &e.Description, &e.Sport.ID, &e.Sport.Name, &e.Sport.IsCustom, &e.StartsAt, &e.EndsAt, &e.Location, &e.Lat, &e.Lng,
		&e.Level, &e.Requirements, &e.PriceNote, &e.PrivateInfo, &e.JoinMode, &e.Capacity,
		&e.ConfirmedCount, &e.PendingCount, &e.WaitlistCount, &e.CompanyMatches,
		&e.Status, &e.ModerationReason, &organizerID, &organizerName, &orgID, &orgName,
		&mpID, &mpStatus, &mpReason, &e.CreatedAt,
		&e.MinParticipants, &e.QuorumDeadline, &e.QuorumStatus, &e.Outdoor, &e.FirstTimersCount, &e.BuddiesCount,
		&mpFirstTime, &mpBuddy, &buddyID, &buddyName)
	if err != nil {
		return Event{}, err
	}
	e.IsOrganizer = organizerID == a.ID
	if orgID != nil {
		e.Organizer = Organizer{Type: "organization", ID: *orgID, Name: orgName}
	} else {
		e.Organizer = Organizer{Type: "user", ID: organizerID, Name: organizerName}
	}
	if mpID != nil {
		e.MyParticipation = &MyParticipation{ID: *mpID, Status: *mpStatus, DecisionReason: *mpReason, FirstTime: mpFirstTime, IsBuddy: mpBuddy}
		if buddyID != nil {
			e.MyParticipation.Buddy = &Person{ID: *buddyID, Name: buddyName}
		}
	}
	if e.Capacity != nil {
		left := max(*e.Capacity-e.ConfirmedCount, 0)
		e.SpotsLeft = &left
	}
	// Точку сбора и контакты видят только те, кто точно придёт: организатор и подтверждённые.
	e.HasPrivateInfo = e.PrivateInfo != ""
	if !e.IsOrganizer && !a.Admin && (e.MyParticipation == nil || e.MyParticipation.Status != "confirmed") {
		e.PrivateInfo = ""
	}
	return e, nil
}

func (r *Repository) GetEvent(ctx context.Context, a Actor, id string) (Event, error) {
	return getEvent(ctx, r.db, a, id)
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func getEvent(ctx context.Context, q querier, a Actor, id string) (Event, error) {
	e, err := scanEvent(q.QueryRow(ctx, eventSelect+" WHERE e.id=$3"+visibleEvent, a.ID, a.Admin, id), a)
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	if err == nil && e.MyParticipation != nil && e.MyParticipation.IsBuddy {
		e.MyParticipation.Newcomers, err = newcomersOf(ctx, q, e.MyParticipation.ID)
	}
	return e, err
}

func (r *Repository) ListEvents(ctx context.Context, a Actor, f EventFilter) (Page[Event], error) {
	page := Page[Event]{Items: []Event{}, Limit: f.Limit, Offset: f.Offset}
	args := []any{a.ID}
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	where := []string{"TRUE"}
	order := " ORDER BY e.starts_at DESC, e.id"
	if len(f.Statuses) == 0 {
		// Пользовательская выдача: только опубликованные и ещё не закончившиеся.
		where = append(where, "e.status='published'", "COALESCE(e.ends_at, e.starts_at) > now()")
		order = " ORDER BY e.starts_at, e.id"
	} else {
		where = append(where, "e.status = ANY("+arg(f.Statuses)+")")
	}
	if f.OrganizerID != "" {
		where = append(where, "e.organizer_id = "+arg(f.OrganizerID))
	}
	if f.OrganizationID != "" {
		where = append(where, "e.organization_id = "+arg(f.OrganizationID)+"::uuid")
	}
	if len(f.SportIDs) > 0 {
		where = append(where, "e.sport_id = ANY("+arg(f.SportIDs)+"::uuid[])")
	}
	if f.MatchedFor != "" {
		// Только события, подобранные под мой запрос «Ищу компанию»: чужой запрос даёт пустой список.
		where = append(where, "e.id IN (SELECT m.event_id FROM company_matches m JOIN company_requests cr ON cr.id=m.request_id WHERE m.request_id="+arg(f.MatchedFor)+"::uuid AND cr.user_id=$1)")
	}
	if g := f.Geo; g != nil {
		if g.BBox != nil {
			where = append(where, "e.lng BETWEEN "+arg(g.BBox[0])+" AND "+arg(g.BBox[2])+" AND e.lat BETWEEN "+arg(g.BBox[1])+" AND "+arg(g.BBox[3]))
		} else {
			dist := distanceSQL(arg(g.Lat), arg(g.Lng))
			where = append(where, "e.lat IS NOT NULL AND "+dist+" <= "+arg(g.RadiusKm))
			order = " ORDER BY " + dist + ", e.starts_at, e.id"
		}
	}
	if f.Mode != "" {
		where = append(where, "e.join_mode = "+arg(f.Mode))
	}
	if f.Level != "" && f.Level != "any" {
		// «Для новичков» — это и события без требований к уровню.
		where = append(where, "e.level IN ('any', "+arg(f.Level)+")")
	}
	if f.From != nil {
		where = append(where, "e.starts_at >= "+arg(*f.From))
	}
	if f.To != nil {
		where = append(where, "e.starts_at < "+arg(*f.To))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		// Полнотекстовый поиск ловит словоформы («пробежки» → «пробежка»), ILIKE — начало слова
		// и названия направления и организации, которых нет в тексте события.
		like := arg("%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q) + "%")
		where = append(where, "(e.search @@ websearch_to_tsquery('russian', "+arg(q)+") OR e.title ILIKE "+like+" OR s.name ILIKE "+like+" OR o.name ILIKE "+like+")")
	}
	cond := " WHERE " + strings.Join(where, " AND ")
	// В счётчике то же соединение, что и в выборке: плейсхолдеры остаются с теми же номерами.
	if err := r.db.QueryRow(ctx, "SELECT count(*) "+eventSelect[strings.Index(eventSelect, "FROM events e"):]+cond, args...).Scan(&page.Total); err != nil {
		return page, err
	}
	rows, err := r.db.Query(ctx, eventSelect+cond+order+" LIMIT "+arg(f.Limit)+" OFFSET "+arg(f.Offset), args...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		e, err := scanEvent(rows, a)
		if err != nil {
			return page, err
		}
		if g := f.Geo; g != nil && g.BBox == nil && e.Lat != nil {
			d := math.Round(DistanceKm(g.Lat, g.Lng, *e.Lat, *e.Lng)*10) / 10
			e.DistanceKm = &d
		}
		page.Items = append(page.Items, e)
	}
	return page, rows.Err()
}

// CreateEvent публикует событие сразу. От имени организации — только если она допущена:
// проверка в той же транзакции, что и вставка, поэтому скрытая кнопка в клиенте не нужна для защиты.
func (r *Repository) CreateEvent(ctx context.Context, a Actor, in EventInput) (Event, error) {
	id := uuid.NewString()
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		sportID, err := resolveSport(ctx, tx, a, in.SportID, in.SportName)
		if err != nil {
			return err
		}
		var orgID *string
		if in.AsOrganization {
			var oid string
			err := tx.QueryRow(ctx, "SELECT id FROM organizations WHERE owner_id=$1 AND status='approved'", a.ID).Scan(&oid)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOrgNotApproved
			}
			if err != nil {
				return err
			}
			orgID = &oid
		}
		_, err = tx.Exec(ctx, `INSERT INTO events(id,organizer_id,organization_id,sport_id,title,description,starts_at,ends_at,location,lat,lng,
level,requirements,price_note,private_info,join_mode,capacity,min_participants,quorum_deadline,quorum_status,outdoor)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
			id, a.ID, orgID, sportID, in.Title, in.Description, in.StartsAt, in.EndsAt, in.Location, in.Lat, in.Lng,
			in.Level, in.Requirements, in.PriceNote, in.PrivateInfo, in.JoinMode, in.Capacity,
			in.MinParticipants, in.QuorumDeadline, quorumStatusFor(in.MinParticipants), in.Outdoor)
		if err != nil {
			return err
		}
		if err = record(ctx, tx, "event", id, platform.Transition{From: "none", To: "published", Actor: actorLabel(a)}); err != nil {
			return err
		}
		return matchNewEvent(ctx, tx, id)
	})
	if err != nil {
		return Event{}, err
	}
	return r.GetEvent(ctx, a, id)
}

// resolveSport возвращает направление из справочника или заводит своё. Повтор с другим
// регистром или пробелами попадает в существующую запись (уникальный индекс по lower(btrim)).
func resolveSport(ctx context.Context, tx pgx.Tx, a Actor, sportID, sportName string) (string, error) {
	var id string
	var hidden bool
	if sportID != "" {
		err := tx.QueryRow(ctx, "SELECT id,hidden FROM sports WHERE id=$1", sportID).Scan(&id, &hidden)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", FieldError{"sport", "Неизвестное направление"}
		}
		if err == nil && hidden {
			return "", ErrSportHidden
		}
		return id, err
	}
	err := tx.QueryRow(ctx, `INSERT INTO sports(id,name,is_custom,created_by) VALUES($1,$2,true,$3)
ON CONFLICT (lower(btrim(name))) DO UPDATE SET name=sports.name RETURNING id,hidden`, uuid.NewString(), sportName, a.ID).Scan(&id, &hidden)
	if err == nil && hidden {
		return "", ErrSportHidden
	}
	return id, err
}

// lockedEvent — поля события, по которым решаются запись и модерация.
type lockedEvent struct {
	OrganizerID, Status, JoinMode, Title, QuorumStatus string
	StartsAt                                           time.Time
	EndsAt, QuorumDeadline                             *time.Time
	Capacity, MinParticipants                          *int
}

func lockEvent(ctx context.Context, tx pgx.Tx, id string) (lockedEvent, error) {
	var e lockedEvent
	err := tx.QueryRow(ctx, "SELECT organizer_id,status,join_mode,title,starts_at,ends_at,capacity,min_participants,quorum_status,quorum_deadline FROM events WHERE id=$1 FOR UPDATE", id).
		Scan(&e.OrganizerID, &e.Status, &e.JoinMode, &e.Title, &e.StartsAt, &e.EndsAt, &e.Capacity, &e.MinParticipants, &e.QuorumStatus, &e.QuorumDeadline)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

func confirmedCount(ctx context.Context, tx pgx.Tx, eventID string) (int, error) {
	var n int
	err := tx.QueryRow(ctx, "SELECT count(*) FROM participations WHERE event_id=$1 AND status='confirmed'", eventID).Scan(&n)
	return n, err
}

// Join записывает пользователя. Свободная запись — сразу «участвую» (или лист ожидания, если мест
// нет), по заявке — «ожидает решения». Снятое, отменённое или прошедшее событие отвечает 409
// даже при переходе по прямой ссылке. Повторный вызов возвращает текущую запись.
func (r *Repository) Join(ctx context.Context, a Actor, eventID, message string, firstTime bool) (MyParticipation, error) {
	var out MyParticipation
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		e, err := lockEvent(ctx, tx, eventID)
		if err != nil {
			return err
		}
		switch {
		case e.Status != "published":
			return ErrEventClosed
		case !endOf(e).After(time.Now()):
			return ErrEventPast
		case e.OrganizerID == a.ID:
			return ErrOwnEvent
		}
		from := "none"
		var existingID string
		err = tx.QueryRow(ctx, "SELECT id,status FROM participations WHERE event_id=$1 AND user_id=$2 FOR UPDATE", eventID, a.ID).Scan(&existingID, &from)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		// Отказ или исключение — решение организатора: иначе его отменил бы повторный клик «Записаться».
		if from == "rejected" {
			return ErrRejected
		}
		if from == "confirmed" || from == "pending" || from == "waitlist" {
			out = MyParticipation{ID: existingID, Status: from}
			return tx.QueryRow(ctx, "SELECT first_time,buddy FROM participations WHERE id=$1", existingID).Scan(&out.FirstTime, &out.IsBuddy)
		}
		to := "pending"
		if e.JoinMode == "open" {
			to = "confirmed"
			if e.Capacity != nil {
				n, err := confirmedCount(ctx, tx, eventID)
				if err != nil {
					return err
				}
				if n >= *e.Capacity {
					to = "waitlist"
				}
			}
		}
		if err = tx.QueryRow(ctx, `INSERT INTO participations(id,event_id,user_id,status,message,first_time) VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT (event_id,user_id) DO UPDATE SET status=EXCLUDED.status, message=EXCLUDED.message, first_time=EXCLUDED.first_time,
  buddy=false, buddy_id=NULL, decision_reason='', decided_by=NULL, decided_at=NULL, updated_at=now()
RETURNING id`, uuid.NewString(), eventID, a.ID, to, message, firstTime).Scan(&out.ID); err != nil {
			return err
		}
		out.Status, out.FirstTime = to, firstTime
		if err = record(ctx, tx, "participation", out.ID, platform.Transition{From: from, To: to, Actor: actorLabel(a), Comment: message}); err != nil {
			return err
		}
		title := map[string]string{"confirmed": "Новый участник", "pending": "Новая заявка на участие", "waitlist": "Новый участник в листе ожидания"}[to]
		who := displayName(a)
		if firstTime {
			who += " (идёт впервые)"
		}
		if err = notify(ctx, tx, e.OrganizerID, title, who+" — «"+e.Title+"»", "/events/"+eventID); err != nil {
			return err
		}
		if to != "confirmed" {
			return nil
		}
		return afterConfirmed(ctx, tx, eventID, out.ID, e)
	})
	return out, err
}

func endOf(e lockedEvent) time.Time {
	if e.EndsAt != nil {
		return *e.EndsAt
	}
	return e.StartsAt
}

// Leave отменяет запись или заявку. Освободившееся место при свободной записи сразу получает
// первый из листа ожидания — организатору ничего делать не нужно.
func (r *Repository) Leave(ctx context.Context, a Actor, eventID string) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		e, err := lockEvent(ctx, tx, eventID)
		if err != nil {
			return err
		}
		var pid, from string
		err = tx.QueryRow(ctx, "SELECT id,status FROM participations WHERE event_id=$1 AND user_id=$2 AND status IN ('confirmed','pending','waitlist') FOR UPDATE", eventID, a.ID).Scan(&pid, &from)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotParticipant
		}
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "UPDATE participations SET status='cancelled', buddy=false, buddy_id=NULL, updated_at=now() WHERE id=$1", pid); err != nil {
			return err
		}
		// Ушёл проводник — его новичков передаём другим, а если некому, предупреждаем организатора.
		if err = releaseNewcomers(ctx, tx, eventID, pid, e); err != nil {
			return err
		}
		if err = record(ctx, tx, "participation", pid, platform.Transition{From: from, To: "cancelled", Actor: actorLabel(a)}); err != nil {
			return err
		}
		if err = notify(ctx, tx, e.OrganizerID, "Участник отменил запись", displayName(a)+" — «"+e.Title+"»", "/events/"+eventID); err != nil {
			return err
		}
		if from != "confirmed" || e.Status != "published" {
			return nil
		}
		return promoteWaitlist(ctx, tx, eventID, e)
	})
}

// Под блокировкой события свободные места сначала получает очередь.
func promoteWaitlist(ctx context.Context, tx pgx.Tx, eventID string, e lockedEvent) error {
	if e.JoinMode != "open" || e.Status != "published" || !endOf(e).After(time.Now()) {
		return nil
	}
	for {
		n, err := confirmedCount(ctx, tx, eventID)
		if err != nil {
			return err
		}
		if e.Capacity != nil && n >= *e.Capacity {
			return nil
		}
		var nextID, nextUser string
		err = tx.QueryRow(ctx, "SELECT id,user_id FROM participations WHERE event_id=$1 AND status='waitlist' ORDER BY created_at, id LIMIT 1 FOR UPDATE", eventID).Scan(&nextID, &nextUser)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "UPDATE participations SET status='confirmed', updated_at=now() WHERE id=$1", nextID); err != nil {
			return err
		}
		if err = record(ctx, tx, "participation", nextID, platform.Transition{From: "waitlist", To: "confirmed", Actor: "system", Comment: "Освободилось место"}); err != nil {
			return err
		}
		if err = notify(ctx, tx, nextUser, "Освободилось место — вы участвуете", "«"+e.Title+"»: запись подтверждена из листа ожидания", "/events/"+eventID); err != nil {
			return err
		}
		if err = afterConfirmed(ctx, tx, eventID, nextID, e); err != nil {
			return err
		}
	}
}

const participantSelect = `SELECT p.id,p.user_id,COALESCE(pr.display_name,''),COALESCE(pr.level,''),p.status,p.message,p.decision_reason,p.created_at,
p.first_time,p.buddy,bp.user_id,COALESCE(bpr.display_name,'')
FROM participations p LEFT JOIN profiles pr ON pr.user_id=p.user_id
LEFT JOIN participations bp ON bp.id=p.buddy_id LEFT JOIN profiles bpr ON bpr.user_id=bp.user_id`

func scanParticipant(row pgx.Row) (Participant, error) {
	var p Participant
	var buddyID *string
	var buddyName string
	err := row.Scan(&p.ID, &p.User.ID, &p.User.Name, &p.User.Level, &p.Status, &p.Message, &p.DecisionReason, &p.CreatedAt,
		&p.FirstTime, &p.IsBuddy, &buddyID, &buddyName)
	if buddyID != nil {
		p.Buddy = &Person{ID: *buddyID, Name: buddyName}
	}
	return p, err
}

// Participants: организатор и админ видят всех и сообщения к заявкам; остальные — только
// подтверждённых участников по имени («кто собирается» — часть ценности продукта).
func (r *Repository) Participants(ctx context.Context, a Actor, eventID string) ([]Participant, error) {
	e, err := r.GetEvent(ctx, a, eventID)
	if err != nil {
		return nil, err
	}
	full := e.IsOrganizer || a.Admin
	query := participantSelect + " WHERE p.event_id=$1 AND p.status='confirmed' ORDER BY p.created_at"
	if full {
		query = participantSelect + ` WHERE p.event_id=$1 ORDER BY
CASE p.status WHEN 'pending' THEN 0 WHEN 'confirmed' THEN 1 WHEN 'waitlist' THEN 2 ELSE 3 END, p.created_at`
	}
	rows, err := r.db.Query(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Participant, error) { return scanParticipant(row) })
	if err != nil {
		return nil, err
	}
	if !full {
		// Пары «новичок — проводник» видит только организатор; остальным достаточно отметок.
		for i := range out {
			out[i].Message, out[i].Buddy = "", nil
		}
	}
	return out, nil
}

// Decide — решение по заявке. Принимает его только организатор события (админ не заменяет
// организатора): проверка organizer_id идёт по заблокированной строке события.
func (r *Repository) Decide(ctx context.Context, a Actor, eventID, pid string, approve bool, reason string) (Participant, error) {
	var out Participant
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		e, err := lockEvent(ctx, tx, eventID)
		if err != nil {
			return err
		}
		if e.OrganizerID != a.ID {
			return ErrNotOrganizer
		}
		if e.Status != "published" {
			return ErrEventClosed
		}
		var from, userID string
		err = tx.QueryRow(ctx, "SELECT status,user_id FROM participations WHERE id=$1 AND event_id=$2 FOR UPDATE", pid, eventID).Scan(&from, &userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if from != "pending" {
			return fmt.Errorf("%w: заявка уже рассмотрена", platform.ErrTransition)
		}
		to := "rejected"
		if approve {
			to = "confirmed"
			if e.Capacity != nil {
				n, err := confirmedCount(ctx, tx, eventID)
				if err != nil {
					return err
				}
				if n >= *e.Capacity {
					return ErrFull
				}
			}
		}
		if _, err = tx.Exec(ctx, "UPDATE participations SET status=$2, decision_reason=$3, decided_by=$4, decided_at=now(), updated_at=now() WHERE id=$1", pid, to, reason, a.ID); err != nil {
			return err
		}
		if err = record(ctx, tx, "participation", pid, platform.Transition{From: from, To: to, Actor: actorLabel(a), Comment: reason}); err != nil {
			return err
		}
		title, body := "Заявка принята", "«"+e.Title+"»: вы в составе участников. Подробности — на странице мероприятия."
		if !approve {
			title, body = "Заявка отклонена", "«"+e.Title+"»"
			if reason != "" {
				body += ": " + reason
			}
		}
		if err = notify(ctx, tx, userID, title, body, "/events/"+eventID); err != nil {
			return err
		}
		if approve {
			if err = afterConfirmed(ctx, tx, eventID, pid, e); err != nil {
				return err
			}
		}
		out, err = scanParticipant(tx.QueryRow(ctx, participantSelect+" WHERE p.id=$1", pid))
		return err
	})
	return out, err
}

// CancelEvent — отмена организатором (причина обязательна: её увидят записавшиеся).
func (r *Repository) CancelEvent(ctx context.Context, a Actor, id, reason string) (Event, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		e, err := lockEvent(ctx, tx, id)
		if err != nil {
			return err
		}
		if e.OrganizerID != a.ID {
			return ErrNotFound
		}
		return changeEventStatus(ctx, tx, a, id, e, "cancelled", reason, "Мероприятие отменено организатором")
	})
	if err != nil {
		return Event{}, err
	}
	return r.GetEvent(ctx, a, id)
}

// ModerateEvent — решение админа: снять с публикации, удалить или вернуть.
// Существующие записи сохраняются: участники и организатор видят плашку с причиной.
func (r *Repository) ModerateEvent(ctx context.Context, a Actor, id, to, reason string) (Event, error) {
	titles := map[string]string{
		"unpublished": "Мероприятие снято с публикации",
		"deleted":     "Мероприятие удалено модератором",
		"published":   "Мероприятие снова опубликовано",
	}
	if titles[to] == "" {
		return Event{}, fmt.Errorf("%w: недопустимое действие", platform.ErrTransition)
	}
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		e, err := lockEvent(ctx, tx, id)
		if err != nil {
			return err
		}
		if to == "published" {
			reason = ""
		}
		return changeEventStatus(ctx, tx, a, id, e, to, reason, titles[to])
	})
	if err != nil {
		return Event{}, err
	}
	return r.GetEvent(ctx, a, id)
}

var eventClosedNote = map[string]string{"unpublished": "Мероприятие снято с публикации: ", "deleted": "Мероприятие удалено: ", "cancelled": "Организатор отменил мероприятие: "}

func changeEventStatus(ctx context.Context, tx pgx.Tx, a Actor, id string, e lockedEvent, to, reason, title string) error {
	if err := eventWorkflow.Check(e.Status, to, reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "UPDATE events SET status=$2, moderation_reason=$3, moderated_by=$4, moderated_at=now(), updated_at=now() WHERE id=$1", id, to, reason, actorLabel(a)); err != nil {
		return err
	}
	if err := record(ctx, tx, "event", id, platform.Transition{From: e.Status, To: to, Actor: actorLabel(a), Comment: reason}); err != nil {
		return err
	}
	// Снятое, удалённое или отменённое событие больше не в выдаче: жалобы на него закрыты.
	if to != "published" {
		if err := closeReports(ctx, tx, a, "event", id, "resolved", eventClosedNote[to]+reason); err != nil {
			return err
		}
	}
	body := "«" + e.Title + "»"
	if reason != "" {
		body += ": " + reason
	}
	recipients := []string{}
	if e.OrganizerID != a.ID {
		recipients = append(recipients, e.OrganizerID)
	}
	rows, err := tx.Query(ctx, "SELECT user_id FROM participations WHERE event_id=$1 AND status IN ('confirmed','pending','waitlist')", id)
	if err != nil {
		return err
	}
	users, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, u := range append(recipients, users...) {
		if err := notify(ctx, tx, u, title, body, "/events/"+id); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) MyParticipations(ctx context.Context, a Actor) ([]MyEntry, error) {
	rows, err := r.db.Query(ctx, `SELECT p.id,p.status,p.decision_reason,e.id,e.title,s.name,e.starts_at,e.location,e.join_mode,e.status,e.moderation_reason,
p.first_time,p.buddy,bp.user_id,COALESCE(bpr.display_name,'')
FROM participations p JOIN events e ON e.id=p.event_id JOIN sports s ON s.id=e.sport_id
LEFT JOIN participations bp ON bp.id=p.buddy_id LEFT JOIN profiles bpr ON bpr.user_id=bp.user_id
WHERE p.user_id=$1 ORDER BY (e.starts_at < now()), e.starts_at LIMIT 200`, a.ID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (MyEntry, error) {
		var m MyEntry
		var buddyID *string
		var buddyName string
		err := row.Scan(&m.ID, &m.Status, &m.DecisionReason, &m.Event.ID, &m.Event.Title, &m.Event.Sport, &m.Event.StartsAt, &m.Event.Location, &m.Event.JoinMode, &m.Event.Status, &m.Event.ModerationReason,
			&m.FirstTime, &m.IsBuddy, &buddyID, &buddyName)
		if buddyID != nil {
			m.Buddy = &Person{ID: *buddyID, Name: buddyName}
		}
		return m, err
	})
}

// ── Обсуждение ──────────────────────────────────────────────────────────────

const commentSelect = `SELECT c.id,c.event_id,c.parent_id,c.author_id,COALESCE(pr.display_name,''),COALESCE(pr.level,''),
c.author_id=e.organizer_id,CASE WHEN c.author_id=e.organizer_id THEN COALESCE(o.name,'') ELSE '' END,
c.body,c.status,c.hidden_reason,c.created_at
FROM comments c JOIN events e ON e.id=c.event_id
LEFT JOIN organizations o ON o.id=e.organization_id
LEFT JOIN profiles pr ON pr.user_id=c.author_id`

func scanComment(row pgx.Row, a Actor) (Comment, error) {
	var c Comment
	err := row.Scan(&c.ID, &c.EventID, &c.ParentID, &c.Author.ID, &c.Author.Name, &c.Author.Level, &c.IsOrganizer, &c.OrganizationName, &c.Body, &c.Status, &c.HiddenReason, &c.CreatedAt)
	// Скрытое модератором или удалённое автором: остальным видна только пометка, текст — админу.
	if err == nil && c.Status != "visible" && !a.Admin {
		c.Body = ""
	}
	return c, err
}

func (r *Repository) Comments(ctx context.Context, a Actor, eventID string) ([]Comment, error) {
	if _, err := r.GetEvent(ctx, a, eventID); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, commentSelect+" WHERE c.event_id=$1 ORDER BY c.created_at, c.id", eventID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Comment, error) { return scanComment(row, a) })
}

// AddComment: вопрос может задать любой вошедший пользователь, пока событие опубликовано.
// Организатор получает уведомление о вопросе, автор вопроса — об ответе; открытые страницы
// обсуждения обновляются через realtime.
func (r *Repository) AddComment(ctx context.Context, a Actor, eventID string, parentID *string, body string) (Comment, error) {
	id := uuid.NewString()
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var status, organizerID, title string
		err := tx.QueryRow(ctx, "SELECT status,organizer_id,title FROM events WHERE id=$1", eventID).Scan(&status, &organizerID, &title)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != "published" {
			return ErrEventClosed
		}
		parentAuthor := ""
		if parentID != nil {
			var grand *string
			err := tx.QueryRow(ctx, "SELECT author_id,parent_id FROM comments WHERE id=$1 AND event_id=$2", *parentID, eventID).Scan(&parentAuthor, &grand)
			if errors.Is(err, pgx.ErrNoRows) || err == nil && grand != nil {
				return ErrBadParent
			}
			if err != nil {
				return err
			}
		}
		if _, err = tx.Exec(ctx, "INSERT INTO comments(id,event_id,author_id,parent_id,body) VALUES($1,$2,$3,$4,$5)", id, eventID, a.ID, parentID, body); err != nil {
			return err
		}
		if err = record(ctx, tx, "comment", id, platform.Transition{From: "none", To: "visible", Actor: actorLabel(a)}); err != nil {
			return err
		}
		link := "/events/" + eventID + "#discussion"
		if a.ID != organizerID {
			if err = notify(ctx, tx, organizerID, "Новый вопрос по мероприятию", displayName(a)+" — «"+title+"»", link); err != nil {
				return err
			}
		}
		if parentAuthor != "" && parentAuthor != a.ID && parentAuthor != organizerID {
			if err = notify(ctx, tx, parentAuthor, "Ответ в обсуждении", displayName(a)+" — «"+title+"»", link); err != nil {
				return err
			}
		}
		return publishChanged(ctx, tx, a.ID, eventID)
	})
	if err != nil {
		return Comment{}, err
	}
	return scanComment(r.db.QueryRow(ctx, commentSelect+" WHERE c.id=$1", id), a)
}

// HideComment скрывает одно сообщение; событие и остальное обсуждение остаются.
func (r *Repository) HideComment(ctx context.Context, a Actor, id, reason string) (Comment, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var eventID, author, status, title string
		err := tx.QueryRow(ctx, "SELECT c.event_id,c.author_id,c.status,e.title FROM comments c JOIN events e ON e.id=c.event_id WHERE c.id=$1 FOR UPDATE OF c", id).Scan(&eventID, &author, &status, &title)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != "visible" {
			return fmt.Errorf("%w: сообщение уже скрыто", platform.ErrTransition)
		}
		if _, err = tx.Exec(ctx, "UPDATE comments SET status='hidden', hidden_reason=$2, hidden_by=$3 WHERE id=$1", id, reason, actorLabel(a)); err != nil {
			return err
		}
		if err = record(ctx, tx, "comment", id, platform.Transition{From: "visible", To: "hidden", Actor: actorLabel(a), Comment: reason}); err != nil {
			return err
		}
		if err = closeReports(ctx, tx, a, "comment", id, "resolved", "Сообщение скрыто: "+reason); err != nil {
			return err
		}
		if err = notify(ctx, tx, author, "Ваше сообщение скрыто модератором", "«"+title+"»: "+reason, "/events/"+eventID+"#discussion"); err != nil {
			return err
		}
		return publishChanged(ctx, tx, a.ID, eventID)
	})
	if err != nil {
		return Comment{}, err
	}
	return scanComment(r.db.QueryRow(ctx, commentSelect+" WHERE c.id=$1", id), a)
}

// ── Организации ─────────────────────────────────────────────────────────────

const orgSelect = `SELECT o.id,o.name,o.description,o.city,o.contacts,o.status,o.review_reason,o.reviewed_at,o.owner_id,
COALESCE(pr.display_name,''),o.created_at,
(SELECT COALESCE(json_agg(json_build_object('id',s.id,'name',s.name,'is_custom',s.is_custom) ORDER BY s.name),'[]')
 FROM organization_sports os JOIN sports s ON s.id=os.sport_id WHERE os.organization_id=o.id),
(SELECT count(*) FROM events e WHERE e.organization_id=o.id AND e.status='published' AND COALESCE(e.ends_at,e.starts_at) > now())
FROM organizations o LEFT JOIN profiles pr ON pr.user_id=o.owner_id`

func scanOrg(row pgx.Row, a Actor) (Organization, error) {
	var o Organization
	var owner Person
	var sports []byte
	if err := row.Scan(&o.ID, &o.Name, &o.Description, &o.City, &o.Contacts, &o.Status, &o.ReviewReason, &o.ReviewedAt, &owner.ID, &owner.Name, &o.CreatedAt, &sports, &o.EventsCount); err != nil {
		return o, err
	}
	if err := json.Unmarshal(sports, &o.Sports); err != nil {
		return o, err
	}
	// Представителя и причину отказа видят сам представитель и админ.
	if a.Admin || owner.ID == a.ID {
		o.Owner = &owner
	} else {
		o.ReviewReason = ""
	}
	return o, nil
}

func (r *Repository) MyOrganization(ctx context.Context, a Actor) (Organization, error) {
	o, err := scanOrg(r.db.QueryRow(ctx, orgSelect+" WHERE o.owner_id=$1", a.ID), a)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNotFound
	}
	return o, err
}

func setOrgSports(ctx context.Context, tx pgx.Tx, orgID string, ids []string) error {
	if _, err := tx.Exec(ctx, "DELETE FROM organization_sports WHERE organization_id=$1", orgID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, "INSERT INTO organization_sports(organization_id,sport_id) SELECT $1, id FROM sports WHERE id = ANY($2::uuid[]) AND NOT hidden", orgID, ids)
	return err
}

// SubmitOrganization создаёт профиль организации со статусом «на рассмотрении».
func (r *Repository) SubmitOrganization(ctx context.Context, a Actor, in OrganizationInput) (Organization, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		id := uuid.NewString()
		tag, err := tx.Exec(ctx, `INSERT INTO organizations(id,owner_id,name,description,city,contacts) VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT (owner_id) DO NOTHING`, id, a.ID, in.Name, in.Description, in.City, in.Contacts)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrOrgExists
		}
		if err = setOrgSports(ctx, tx, id, in.SportIDs); err != nil {
			return err
		}
		return record(ctx, tx, "organization", id, platform.Transition{From: "none", To: "pending", Actor: actorLabel(a)})
	})
	if err != nil {
		return Organization{}, err
	}
	return r.MyOrganization(ctx, a)
}

// UpdateOrganization правит профиль. После отказа правка означает повторную подачу: статус
// возвращается в «на рассмотрении», прежняя причина остаётся в истории.
func (r *Repository) UpdateOrganization(ctx context.Context, a Actor, in OrganizationInput) (Organization, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var id, status string
		err := tx.QueryRow(ctx, "SELECT id,status FROM organizations WHERE owner_id=$1 FOR UPDATE", a.ID).Scan(&id, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		next := status
		if status == "rejected" {
			next = "pending"
			if err = orgWorkflow.Check(status, next, ""); err != nil {
				return err
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE organizations SET name=$2,description=$3,city=$4,contacts=$5,status=$6,
review_reason=CASE WHEN $6='pending' THEN '' ELSE review_reason END, updated_at=now() WHERE id=$1`,
			id, in.Name, in.Description, in.City, in.Contacts, next); err != nil {
			return err
		}
		if err = setOrgSports(ctx, tx, id, in.SportIDs); err != nil {
			return err
		}
		comment := "Профиль организации изменён"
		if status != next {
			comment = "Профиль исправлен и подан повторно"
		}
		return record(ctx, tx, "organization", id, platform.Transition{From: status, To: next, Actor: actorLabel(a), Comment: comment})
	})
	if err != nil {
		return Organization{}, err
	}
	return r.MyOrganization(ctx, a)
}

// ListOrganizations: без статусов — публичный каталог (только допущенные), со статусами — админ.
func (r *Repository) ListOrganizations(ctx context.Context, a Actor, q string, statuses []string, limit, offset int) (Page[Organization], error) {
	page := Page[Organization]{Items: []Organization{}, Limit: limit, Offset: offset}
	order := " ORDER BY lower(o.name), o.id"
	if len(statuses) == 0 {
		statuses = []string{"approved"}
	} else {
		order = " ORDER BY CASE o.status WHEN 'pending' THEN 0 ELSE 1 END, o.created_at DESC, o.id"
	}
	cond := " WHERE o.status = ANY($1)"
	args := []any{statuses}
	if q = strings.TrimSpace(q); q != "" {
		args = append(args, "%"+strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q)+"%")
		cond += " AND (o.name ILIKE $2 OR o.description ILIKE $2 OR o.city ILIKE $2)"
	}
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM organizations o"+cond, args...).Scan(&page.Total); err != nil {
		return page, err
	}
	n := len(args)
	rows, err := r.db.Query(ctx, orgSelect+cond+order+" LIMIT $"+strconv.Itoa(n+1)+" OFFSET $"+strconv.Itoa(n+2), append(args, limit, offset)...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		o, err := scanOrg(rows, a)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, o)
	}
	return page, rows.Err()
}

// GetOrganization: страница публична только после допуска; представитель и админ видят её всегда.
func (r *Repository) GetOrganization(ctx context.Context, a Actor, id string) (Organization, error) {
	o, err := scanOrg(r.db.QueryRow(ctx, orgSelect+" WHERE o.id=$1 AND (o.status='approved' OR o.owner_id=$2 OR $3::bool)", id, a.ID, a.Admin), a)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNotFound
	}
	return o, err
}

// ReviewOrganization — допуск админом. Отказ без причины невозможен (Workflow и CHECK в базе).
func (r *Repository) ReviewOrganization(ctx context.Context, a Actor, id, to, reason string) (Organization, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var from, owner, name string
		err := tx.QueryRow(ctx, "SELECT status,owner_id,name FROM organizations WHERE id=$1 FOR UPDATE", id).Scan(&from, &owner, &name)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if to != "approved" && to != "rejected" {
			return fmt.Errorf("%w: недопустимое решение", platform.ErrTransition)
		}
		if err = orgWorkflow.Check(from, to, reason); err != nil {
			return err
		}
		if to == "approved" {
			reason = ""
		}
		if _, err = tx.Exec(ctx, "UPDATE organizations SET status=$2, review_reason=$3, reviewed_by=$4, reviewed_at=now(), updated_at=now() WHERE id=$1", id, to, reason, actorLabel(a)); err != nil {
			return err
		}
		if err = record(ctx, tx, "organization", id, platform.Transition{From: from, To: to, Actor: actorLabel(a), Comment: reason}); err != nil {
			return err
		}
		if to == "approved" {
			return notify(ctx, tx, owner, "Организация допущена", "«"+name+"» прошла проверку: теперь можно публиковать мероприятия от её имени.", "/organizations/"+id)
		}
		return notify(ctx, tx, owner, "Заявка организации отклонена", "«"+name+"»: "+reason, "/my/organization")
	})
	if err != nil {
		return Organization{}, err
	}
	return r.GetOrganization(ctx, a, id)
}

// ── Профили ─────────────────────────────────────────────────────────────────

func (r *Repository) GetProfile(ctx context.Context, a Actor, id string) (Profile, error) {
	var p Profile
	err := r.db.QueryRow(ctx, "SELECT user_id,display_name,about,city,COALESCE(level,'') FROM profiles WHERE user_id=$1", id).Scan(&p.ID, &p.Name, &p.About, &p.City, &p.Level)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	page, err := r.ListEvents(ctx, a, EventFilter{OrganizerID: id, Limit: 20})
	if err != nil {
		return p, err
	}
	p.Events = page.Items
	p.Stats, err = r.profileStats(ctx, id)
	return p, err
}

// ── Исходящие события ───────────────────────────────────────────────────────

// notify ставит уведомление в outbox той же транзакции: решение и уведомление о нём
// сохраняются вместе, доставку (колокольчик, MAX) выполняет сервис notifications.
func notify(ctx context.Context, tx pgx.Tx, userID, title, body, link string) error {
	if userID == "" {
		return nil
	}
	payload, _ := json.Marshal(map[string]string{"title": truncate(title, 200), "body": truncate(body, 1000), "link": link})
	_, err := tx.Exec(ctx, "INSERT INTO outbox(id,type,user_id,request_id,payload) VALUES($1,'notification.requested',$2,$3,$4)", uuid.NewString(), userID, platform.RequestID(ctx), payload)
	return err
}

// publishChanged сообщает открытым страницам, что обсуждение изменилось. В канал room:
// может писать любой вошедший, поэтому сообщение — только сигнал «перечитай», данные клиент
// берёт из API.
func publishChanged(ctx context.Context, tx pgx.Tx, userID, eventID string) error {
	payload, _ := json.Marshal(map[string]any{
		"channel": "room:event:" + eventID, "from": "events",
		"data": map[string]string{"type": "comments.changed"}, "at": time.Now().UTC(),
	})
	_, err := tx.Exec(ctx, "INSERT INTO outbox(id,type,user_id,request_id,payload) VALUES($1,'realtime.publish',$2,$3,$4)", uuid.NewString(), userID, platform.RequestID(ctx), payload)
	return err
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
