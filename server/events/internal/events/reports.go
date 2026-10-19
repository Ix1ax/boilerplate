package events

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Жалобы, жизненный цикл сообщения и исключение участника — то, без чего модерация держится
// только на том, что админ сам заметит нарушение. Жалоба не меняет данные сама по себе:
// решение остаётся за человеком (админом), а пользователь видит, что его сигнал рассмотрен.

var (
	ErrOwnTarget      = errors.New("Нельзя пожаловаться на своё мероприятие, сообщение, отзыв или историю")
	ErrReportExists   = errors.New("Вы уже пожаловались — жалоба ждёт рассмотрения")
	ErrTooManyReports = errors.New("Слишком много жалоб за час. Если нарушений много, напишите в поддержку")
	ErrNotAuthor      = errors.New("Удалить можно только своё сообщение")
)

// reportReasons — категории жалобы; подписи для людей — в OpenAPI и интерфейсе.
var reportReasons = map[string]bool{"spam": true, "offensive": true, "fraud": true, "wrong_info": true, "dangerous": true, "other": true}

// reportsPerHour — защита от «засыпать жалобами» конкурента: открытые и рассмотренные считаются вместе.
const reportsPerHour = 20

type ReportInput struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

// Report — жалоба. Target — краткое описание цели, чтобы админ решал, не открывая её отдельно.
type Report struct {
	ID         string `json:"id"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	// EventID — мероприятие цели; nil у истории и личного сообщения.
	EventID    *string      `json:"event_id"`
	Reason     string       `json:"reason"`
	Details    string       `json:"details"`
	Status     string       `json:"status"`
	Resolution string       `json:"resolution"`
	Reporter   Person       `json:"reporter"`
	Target     ReportTarget `json:"target"`
	// OpenOnTarget — сколько открытых жалоб на ту же цель: десять жалоб важнее одной.
	OpenOnTarget int        `json:"open_on_target"`
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at"`
}

// ReportTarget — цель глазами админа. Title понятен без типа: «Отзыв о «…»», «История»,
// «Личное сообщение». Status — в терминах цели: visible/hidden/deleted у сообщений и отзывов,
// active/expired/deleted у истории, статус мероприятия у события.
type ReportTarget struct {
	Title  string `json:"title"`
	Text   string `json:"text"`
	Author Person `json:"author"`
	Status string `json:"status"`
	// MediaURL — фото или видео истории по админскому адресу (доступно и после истечения, пока жалоба открыта).
	MediaURL *string `json:"media_url"`
}

// MyReport — что пользователь видит о своих жалобах.
type MyReport struct {
	ID         string  `json:"id"`
	TargetType string  `json:"target_type"`
	TargetID   string  `json:"target_id"`
	EventID    *string `json:"event_id"`
	// EventTitle — название мероприятия; у истории и личного сообщения пусто.
	EventTitle string     `json:"event_title"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
	Resolution string     `json:"resolution"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
}

func validateReport(in *ReportInput) error {
	in.Reason, in.Details = strings.TrimSpace(in.Reason), strings.TrimSpace(in.Details)
	switch {
	case !reportReasons[in.Reason]:
		return FieldError{"reason", "Выберите причину: spam, offensive, fraud, wrong_info, dangerous или other"}
	case utf8.RuneCountInString(in.Details) > 1000:
		return FieldError{"details", "Пояснение — не длиннее 1000 символов"}
	case in.Reason == "other" && in.Details == "":
		return FieldError{"details", "Опишите, что не так: для причины «другое» пояснение обязательно"}
	}
	return nil
}

func (s *Service) Report(ctx context.Context, a Actor, targetType, eventID, targetID string, in ReportInput) (MyReport, error) {
	if err := validateReport(&in); err != nil {
		return MyReport{}, err
	}
	return s.Store.CreateReport(ctx, a, targetType, eventID, targetID, in)
}

func (s *Service) DismissReport(ctx context.Context, a Actor, id, resolution string) (Report, error) {
	resolution, err := cleanReason(resolution)
	if err != nil {
		return Report{}, err
	}
	if resolution == "" {
		return Report{}, FieldError{"resolution", "Поясните решение: его увидит автор жалобы"}
	}
	return s.Store.DismissReport(ctx, a, id, resolution)
}

func (s *Service) RemoveParticipant(ctx context.Context, a Actor, eventID, pid, reason string) (Participant, error) {
	reason, err := cleanReason(reason)
	if err != nil {
		return Participant{}, err
	}
	if reason == "" {
		return Participant{}, FieldError{"reason", "Укажите причину: её увидит участник"}
	}
	return s.Store.RemoveParticipant(ctx, a, eventID, pid, reason)
}

// reportTargetTypes — на что можно пожаловаться. Подписи для людей — в contracts (reportTargetLabels).
var reportTargetTypes = map[string]bool{"event": true, "comment": true, "review": true, "story": true, "direct_message": true}

// CreateReport проверяет цель в той же транзакции, что и запись: жалоба возможна только на то,
// что жалующийся сейчас видит, и не на своё. parentID — мероприятие для event и comment,
// переписка для direct_message; у отзыва и истории родителя в адресе нет.
func (r *Repository) CreateReport(ctx context.Context, a Actor, targetType, parentID, targetID string, in ReportInput) (MyReport, error) {
	if !reportTargetTypes[targetType] {
		return MyReport{}, ErrNotFound
	}
	id := uuid.NewString()
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		eventID, author, err := reportTarget(ctx, tx, a, targetType, parentID, targetID)
		if err != nil {
			return err
		}
		if author == a.ID {
			return ErrOwnTarget
		}
		var recent int
		if err = tx.QueryRow(ctx, "SELECT count(*) FROM reports WHERE reporter_id=$1 AND created_at > now() - interval '1 hour'", a.ID).Scan(&recent); err != nil {
			return err
		}
		if recent >= reportsPerHour {
			return ErrTooManyReports
		}
		_, err = tx.Exec(ctx, "INSERT INTO reports(id,target_type,target_id,event_id,reporter_id,reason,details) VALUES($1,$2,$3,$4,$5,$6,$7)",
			id, targetType, targetID, eventID, a.ID, in.Reason, in.Details)
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return ErrReportExists
		}
		if err != nil {
			return err
		}
		// Создание тоже событие: так moderation узнаёт о жалобе (в журнал оно не попадает).
		return record(ctx, tx, "report", id, platform.Transition{From: "none", To: "open", Actor: actorLabel(a), Comment: in.Reason})
	})
	if err != nil {
		return MyReport{}, err
	}
	return r.myReport(ctx, r.db, a, id)
}

// reportTarget находит цель жалобы с теми же правилами видимости, что у её обычного чтения, и
// возвращает мероприятие (nil у истории и личного сообщения) и автора. Невидимая цель — 404,
// чтобы по ответу нельзя было узнать о чужой истории или переписке.
func reportTarget(ctx context.Context, tx pgx.Tx, a Actor, targetType, parentID, targetID string) (*string, string, error) {
	var eventID *string
	var author string
	var err error
	switch targetType {
	case "event", "comment":
		// Событие должно быть видно жалующемуся (те же правила, что у карточки).
		var e Event
		if e, err = getEvent(ctx, tx, a, parentID); err != nil {
			return nil, "", err
		}
		eventID = &e.ID
		if targetType == "event" {
			if e.IsOrganizer {
				return eventID, a.ID, nil
			}
			return eventID, "", nil
		}
		err = tx.QueryRow(ctx, "SELECT author_id FROM comments WHERE id=$1 AND event_id=$2 AND status='visible'", targetID, parentID).Scan(&author)
	case "review":
		// Отзыв виден в профиле организатора, пока не скрыт и мероприятие опубликовано.
		err = tx.QueryRow(ctx, `SELECT r.event_id::text,r.author_id FROM event_reviews r JOIN events e ON e.id=r.event_id
WHERE r.id=$1 AND r.hidden_at IS NULL AND e.status='published'`, targetID).Scan(&eventID, &author)
	case "story":
		// Историю видят автор и друзья, пока она не удалена и не истекла (как в ленте).
		err = tx.QueryRow(ctx, `SELECT s.author_id FROM stories s WHERE s.id=$1 AND s.deleted_at IS NULL AND s.expires_at>now()
AND (s.author_id=$2 OR EXISTS(SELECT 1 FROM friendships f WHERE f.user_a=LEAST($2,s.author_id) AND f.user_b=GREATEST($2,s.author_id) AND f.status='accepted'))`,
			targetID, a.ID).Scan(&author)
	case "direct_message":
		// Пожаловаться на личное сообщение может только участник переписки.
		err = tx.QueryRow(ctx, `SELECT m.author_id FROM direct_messages m JOIN direct_chats c ON c.id=m.chat_id
WHERE m.id=$1 AND c.id=$2 AND $3 IN(c.user_a,c.user_b) AND m.hidden_at IS NULL`, targetID, parentID, a.ID).Scan(&author)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	return eventID, author, err
}

const myReportSelect = `SELECT r.id,r.target_type,r.target_id,r.event_id::text,COALESCE(e.title,''),r.reason,r.status,r.resolution,r.created_at,r.resolved_at
FROM reports r LEFT JOIN events e ON e.id=r.event_id`

func scanMyReport(row pgx.Row) (MyReport, error) {
	var m MyReport
	err := row.Scan(&m.ID, &m.TargetType, &m.TargetID, &m.EventID, &m.EventTitle, &m.Reason, &m.Status, &m.Resolution, &m.CreatedAt, &m.ResolvedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

func (r *Repository) myReport(ctx context.Context, q querier, a Actor, id string) (MyReport, error) {
	return scanMyReport(q.QueryRow(ctx, myReportSelect+" WHERE r.id=$1 AND r.reporter_id=$2", id, a.ID))
}

func (r *Repository) MyReports(ctx context.Context, a Actor) ([]MyReport, error) {
	rows, err := r.db.Query(ctx, myReportSelect+" WHERE r.reporter_id=$1 ORDER BY r.created_at DESC LIMIT 100", a.ID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (MyReport, error) { return scanMyReport(row) })
}

// reportSelect — очередь для админа: цель описана сразу (заголовок, текст и автор), даже если
// она уже скрыта или удалена — админ видит оригинал. Из личной переписки виден только текст
// того сообщения, на которое пожаловались.
const reportSelect = `SELECT r.id,r.target_type,r.target_id,r.event_id::text,r.reason,r.details,r.status,r.resolution,
r.reporter_id,COALESCE(rp.display_name,''),COALESCE(rp.level,''),
COALESCE(t.title,''),COALESCE(t.body,''),COALESCE(t.author_id,''),COALESCE(ap.display_name,''),COALESCE(t.status,''),
(SELECT count(*) FROM reports x WHERE x.target_type=r.target_type AND x.target_id=r.target_id AND x.status='open'),
r.created_at,r.resolved_at,t.media_url
FROM reports r LEFT JOIN events e ON e.id=r.event_id
LEFT JOIN LATERAL (
  SELECT e.title, left(e.description,500) AS body, e.organizer_id AS author_id, e.status, NULL::text AS media_url WHERE r.target_type='event'
  UNION ALL SELECT e.title, left(c.body,500), c.author_id, c.status, NULL FROM comments c WHERE r.target_type='comment' AND c.id=r.target_id
  UNION ALL SELECT 'Отзыв о «'||COALESCE(e.title,'')||'»', left(v.body,500), v.author_id,
    CASE WHEN v.hidden_at IS NULL THEN 'visible' ELSE 'hidden' END, NULL FROM event_reviews v WHERE r.target_type='review' AND v.id=r.target_id
  UNION ALL SELECT 'История', left(s.body,500), s.author_id,
    CASE WHEN s.deleted_at IS NOT NULL THEN 'deleted' WHEN s.expires_at<=now() THEN 'expired' ELSE 'active' END,
    CASE WHEN s.media_key IS NOT NULL THEN '/api/v1/admin/stories/'||s.id::text||'/media' END FROM stories s WHERE r.target_type='story' AND s.id=r.target_id
  UNION ALL SELECT 'Личное сообщение', left(m.body,500), m.author_id,
    CASE WHEN m.hidden_at IS NULL THEN 'visible' ELSE 'hidden' END, NULL FROM direct_messages m WHERE r.target_type='direct_message' AND m.id=r.target_id
) t ON true
LEFT JOIN profiles rp ON rp.user_id=r.reporter_id
LEFT JOIN profiles ap ON ap.user_id=t.author_id`

func scanReport(row pgx.Row) (Report, error) {
	var x Report
	var level, authorID string
	err := row.Scan(&x.ID, &x.TargetType, &x.TargetID, &x.EventID, &x.Reason, &x.Details, &x.Status, &x.Resolution,
		&x.Reporter.ID, &x.Reporter.Name, &level, &x.Target.Title, &x.Target.Text, &authorID, &x.Target.Author.Name,
		&x.Target.Status, &x.OpenOnTarget, &x.CreatedAt, &x.ResolvedAt, &x.Target.MediaURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return x, ErrNotFound
	}
	x.Reporter.Level, x.Target.Author.ID = level, authorID
	return x, err
}

// AdminReports — очередь: открытые сначала самые «жалуемые» цели, затем старые.
func (r *Repository) AdminReports(ctx context.Context, status, targetType string, limit, offset int) (Page[Report], error) {
	page := Page[Report]{Items: []Report{}, Limit: limit, Offset: offset}
	where := " WHERE ($1='' OR r.status=$1) AND ($2='' OR r.target_type=$2)"
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM reports r"+where, status, targetType).Scan(&page.Total); err != nil {
		return page, err
	}
	// 17 — число открытых жалоб на цель (столбец reportSelect).
	rows, err := r.db.Query(ctx, reportSelect+where+" ORDER BY (r.status<>'open'), 17 DESC, r.created_at, r.id LIMIT $3 OFFSET $4", status, targetType, limit, offset)
	if err != nil {
		return page, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Report, error) { return scanReport(row) })
	if err != nil {
		return page, err
	}
	page.Items = items
	return page, nil
}

// DismissReport закрывает все открытые жалобы на ту же цель: решение принимается о цели, а не
// о каждой жалобе по отдельности. Авторы жалоб получают пояснение.
func (r *Repository) DismissReport(ctx context.Context, a Actor, id, resolution string) (Report, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var targetType, targetID, status string
		err := tx.QueryRow(ctx, "SELECT target_type,target_id,status FROM reports WHERE id=$1 FOR UPDATE", id).Scan(&targetType, &targetID, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != "open" {
			return fmt.Errorf("%w: жалоба уже рассмотрена", platform.ErrTransition)
		}
		return closeReports(ctx, tx, a, targetType, targetID, "dismissed", resolution)
	})
	if err != nil {
		return Report{}, err
	}
	return scanReport(r.db.QueryRow(ctx, reportSelect+" WHERE r.id=$1", id))
}

// closeReports — общий финал: при скрытии или удалении цели (resolved) и при отказе
// (dismissed). Вызывается в транзакции, которая меняет саму цель.
func closeReports(ctx context.Context, tx pgx.Tx, a Actor, targetType, targetID, to, resolution string) error {
	rows, err := tx.Query(ctx, `UPDATE reports r SET status=$3, resolution=$4, resolved_by=$5, resolved_at=now()
WHERE r.target_type=$1 AND r.target_id=$2 AND r.status='open'
RETURNING r.id, r.reporter_id, r.event_id::text, COALESCE((SELECT title FROM events WHERE id=r.event_id),'')`, targetType, targetID, to, resolution, actorLabel(a))
	if err != nil {
		return err
	}
	type closed struct {
		id, reporter, title string
		eventID             *string
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (closed, error) {
		var c closed
		return c, row.Scan(&c.id, &c.reporter, &c.eventID, &c.title)
	})
	if err != nil {
		return err
	}
	title := "Жалоба рассмотрена: меры приняты"
	if to == "dismissed" {
		title = "Жалоба рассмотрена: нарушений не нашли"
	}
	for _, c := range list {
		if err = record(ctx, tx, "report", c.id, platform.Transition{From: "open", To: to, Actor: actorLabel(a), Comment: resolution}); err != nil {
			return err
		}
		body, link := reportSubject(targetType, c.title), ""
		if c.eventID != nil {
			link = "/events/" + *c.eventID
		}
		if resolution != "" {
			body += ": " + resolution
		}
		if err = notify(ctx, tx, c.reporter, title, body, link); err != nil {
			return err
		}
	}
	return nil
}

// reportSubject — о чём была жалоба, словами для уведомления.
func reportSubject(targetType, eventTitle string) string {
	switch targetType {
	case "review":
		return "Отзыв о «" + eventTitle + "»"
	case "story":
		return "История"
	case "direct_message":
		return "Личное сообщение"
	}
	return "«" + eventTitle + "»"
}

// DeleteComment — автор убирает своё сообщение. Текст скрывается у всех, кроме админа
// (на случай жалобы), ответы остаются: обсуждение не рассыпается.
func (r *Repository) DeleteComment(ctx context.Context, a Actor, eventID, id string) (Comment, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var author, status string
		err := tx.QueryRow(ctx, "SELECT author_id,status FROM comments WHERE id=$1 AND event_id=$2 FOR UPDATE", id, eventID).Scan(&author, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if author != a.ID {
			return ErrNotAuthor
		}
		if status != "visible" {
			return fmt.Errorf("%w: сообщение уже скрыто или удалено", platform.ErrTransition)
		}
		if _, err = tx.Exec(ctx, "UPDATE comments SET status='deleted' WHERE id=$1", id); err != nil {
			return err
		}
		if err = record(ctx, tx, "comment", id, platform.Transition{From: "visible", To: "deleted", Actor: actorLabel(a)}); err != nil {
			return err
		}
		// Удалённое сообщение больше не нарушает правила: жалобы на него закрываем.
		if err = closeReports(ctx, tx, a, "comment", id, "resolved", "Автор удалил сообщение"); err != nil {
			return err
		}
		return publishChanged(ctx, tx, a.ID, eventID)
	})
	if err != nil {
		return Comment{}, err
	}
	return scanComment(r.db.QueryRow(ctx, commentSelect+" WHERE c.id=$1", id), a)
}

// RestoreComment — админ возвращает ошибочно скрытое сообщение; автор получает уведомление.
func (r *Repository) RestoreComment(ctx context.Context, a Actor, id string) (Comment, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var eventID, author, status, title string
		err := tx.QueryRow(ctx, "SELECT c.event_id,c.author_id,c.status,e.title FROM comments c JOIN events e ON e.id=c.event_id WHERE c.id=$1 FOR UPDATE OF c", id).Scan(&eventID, &author, &status, &title)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != "hidden" {
			return fmt.Errorf("%w: вернуть можно только скрытое модератором сообщение", platform.ErrTransition)
		}
		if _, err = tx.Exec(ctx, "UPDATE comments SET status='visible', hidden_reason='', hidden_by=NULL WHERE id=$1", id); err != nil {
			return err
		}
		if err = record(ctx, tx, "comment", id, platform.Transition{From: "hidden", To: "visible", Actor: actorLabel(a)}); err != nil {
			return err
		}
		if err = notify(ctx, tx, author, "Ваше сообщение снова видно", "«"+title+"»: модератор вернул сообщение в обсуждение", "/events/"+eventID+"#discussion"); err != nil {
			return err
		}
		return publishChanged(ctx, tx, a.ID, eventID)
	})
	if err != nil {
		return Comment{}, err
	}
	return scanComment(r.db.QueryRow(ctx, commentSelect+" WHERE c.id=$1", id), a)
}

// RemoveParticipant — организатор исключает подтверждённого участника или снимает человека
// из листа ожидания с причиной. Освободившееся место сразу получает очередь.
func (r *Repository) RemoveParticipant(ctx context.Context, a Actor, eventID, pid, reason string) (Participant, error) {
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
		if !endOf(e).After(time.Now()) {
			return ErrEventPast
		}
		var from, userID string
		err = tx.QueryRow(ctx, "SELECT status,user_id FROM participations WHERE id=$1 AND event_id=$2 FOR UPDATE", pid, eventID).Scan(&from, &userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if from != "confirmed" && from != "waitlist" {
			return fmt.Errorf("%w: исключить можно участника или человека из листа ожидания; заявку — отклонить", platform.ErrTransition)
		}
		if _, err = tx.Exec(ctx, "UPDATE participations SET status='rejected', buddy=false, buddy_id=NULL, decision_reason=$2, decided_by=$3, decided_at=now(), updated_at=now() WHERE id=$1", pid, reason, a.ID); err != nil {
			return err
		}
		// После исключения проводника назначаем замену или предупреждаем новичков и организатора.
		if err = releaseNewcomers(ctx, tx, eventID, pid, e); err != nil {
			return err
		}
		if err = record(ctx, tx, "participation", pid, platform.Transition{From: from, To: "rejected", Actor: actorLabel(a), Comment: reason}); err != nil {
			return err
		}
		if err = notify(ctx, tx, userID, "Организатор исключил вас из состава", "«"+e.Title+"»: "+reason, "/events/"+eventID); err != nil {
			return err
		}
		if from == "confirmed" {
			if err = promoteWaitlist(ctx, tx, eventID, e); err != nil {
				return err
			}
		}
		out, err = scanParticipant(tx.QueryRow(ctx, participantSelect+" WHERE p.id=$1", pid))
		return err
	})
	return out, err
}

// AdminReport — одна жалоба с описанием цели для разбора (копилот модератора).
func (r *Repository) AdminReport(ctx context.Context, id string) (Report, error) {
	return scanReport(r.db.QueryRow(ctx, reportSelect+" WHERE r.id=$1", id))
}
