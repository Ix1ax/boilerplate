package moderation

import (
	"context"
	"strconv"
	"time"

	"example.com/hackathon/server/moderation/internal/platform"
	"github.com/jackc/pgx/v5"
)

// Dashboard — первая страница админки: что требует решения и как живёт платформа.
type Dashboard struct {
	Organizations struct {
		Pending  int `json:"pending"`
		Approved int `json:"approved"`
		Rejected int `json:"rejected"`
	} `json:"organizations"`
	Events struct {
		Upcoming    int `json:"upcoming"`
		Unpublished int `json:"unpublished"`
		Deleted     int `json:"deleted"`
		Cancelled   int `json:"cancelled"`
		CreatedWeek int `json:"created_week"`
	} `json:"events"`
	Participations struct {
		Confirmed int `json:"confirmed"`
		Pending   int `json:"pending"`
		FirstTime int `json:"first_time"`
	} `json:"participations"`
	Comments struct {
		Total  int `json:"total"`
		Hidden int `json:"hidden"`
	} `json:"comments"`
	CompanyRequests struct {
		Active int `json:"active"`
	} `json:"company_requests"`
	// Reports — жалобы пользователей: сколько ждут решения и как разобраны за неделю.
	Reports struct {
		Open          int `json:"open"`
		ResolvedWeek  int `json:"resolved_week"`
		DismissedWeek int `json:"dismissed_week"`
		// TopReason — самая частая причина среди открытых жалоб; null, если открытых нет.
		TopReason *string `json:"top_reason"`
	} `json:"reports"`
	// Sports — пользовательские направления («своё направление» при создании события).
	Sports struct {
		Custom int `json:"custom"`
		Hidden int `json:"hidden"`
	} `json:"sports"`
	Quorum struct {
		Gathering int `json:"gathering"`
		Reached   int `json:"reached"`
		Failed    int `json:"failed"`
	} `json:"quorum"`
	// Series — 14 последних дней по Красноярску: новые события и записи.
	Series []DayStat `json:"series"`
	// TopSports — направления с наибольшим числом предстоящих событий: видно, что платформа не про один спорт.
	TopSports []SportStat `json:"top_sports"`
	// PendingOrganizations — самые давние заявки на допуск: их надо рассмотреть первыми.
	PendingOrganizations []PendingOrganization `json:"pending_organizations"`
	// UpdatedAt — время самого свежего полученного изменения: проекция может отставать на секунды.
	UpdatedAt *time.Time `json:"updated_at"`
}

type DayStat struct {
	Date           string `json:"date"`
	Events         int    `json:"events"`
	Participations int    `json:"participations"`
}

type SportStat struct {
	Sport  string `json:"sport"`
	Events int    `json:"events"`
}

type PendingOrganization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OwnerName string    `json:"owner_name"`
	CreatedAt time.Time `json:"created_at"`
}

const localTZ = "Asia/Krasnoyarsk"

// Dashboard читает всё одним снимком (REPEATABLE READ): цифры на экране согласованы между собой.
func (r *Repository) Dashboard(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return d, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `SELECT
(SELECT count(*) FROM organizations WHERE status='pending'),
(SELECT count(*) FROM organizations WHERE status='approved'),
(SELECT count(*) FROM organizations WHERE status='rejected'),
(SELECT count(*) FROM events WHERE status='published' AND starts_at > now()),
(SELECT count(*) FROM events WHERE status='unpublished'),
(SELECT count(*) FROM events WHERE status='deleted'),
(SELECT count(*) FROM events WHERE status='cancelled'),
(SELECT count(*) FROM events WHERE created_at > now() - interval '7 days'),
(SELECT count(*) FROM participations WHERE status='confirmed'),
(SELECT count(*) FROM participations WHERE status='pending'),
(SELECT count(*) FROM participations WHERE status='confirmed' AND first_time),
(SELECT count(*) FROM comments),
(SELECT count(*) FROM comments WHERE status='hidden'),
(SELECT count(*) FROM company_requests WHERE status='active'),
(SELECT count(*) FROM events WHERE status='published' AND quorum_status='gathering' AND starts_at > now()),
(SELECT count(*) FROM events WHERE quorum_status='reached'),
(SELECT count(*) FROM events WHERE quorum_status='failed'),
(SELECT count(*) FROM reports WHERE status='open'),
(SELECT count(*) FROM reports WHERE status='resolved' AND resolved_at > now() - interval '7 days'),
(SELECT count(*) FROM reports WHERE status='dismissed' AND resolved_at > now() - interval '7 days'),
(SELECT reason FROM reports WHERE status='open' GROUP BY reason ORDER BY count(*) DESC, reason LIMIT 1),
(SELECT count(*) FROM sports WHERE is_custom),
(SELECT count(*) FROM sports WHERE is_custom AND hidden),
(SELECT max(m) FROM (SELECT max(changed_at) m FROM events UNION ALL SELECT max(changed_at) FROM organizations
  UNION ALL SELECT max(changed_at) FROM participations UNION ALL SELECT max(changed_at) FROM comments
  UNION ALL SELECT max(changed_at) FROM reports UNION ALL SELECT max(changed_at) FROM sports) x)`).Scan(
		&d.Organizations.Pending, &d.Organizations.Approved, &d.Organizations.Rejected,
		&d.Events.Upcoming, &d.Events.Unpublished, &d.Events.Deleted, &d.Events.Cancelled, &d.Events.CreatedWeek,
		&d.Participations.Confirmed, &d.Participations.Pending, &d.Participations.FirstTime,
		&d.Comments.Total, &d.Comments.Hidden, &d.CompanyRequests.Active,
		&d.Quorum.Gathering, &d.Quorum.Reached, &d.Quorum.Failed,
		&d.Reports.Open, &d.Reports.ResolvedWeek, &d.Reports.DismissedWeek, &d.Reports.TopReason,
		&d.Sports.Custom, &d.Sports.Hidden, &d.UpdatedAt)
	if err != nil {
		return d, err
	}
	rows, err := tx.Query(ctx, `SELECT to_char(day, 'YYYY-MM-DD'),
(SELECT count(*) FROM events e WHERE (e.created_at AT TIME ZONE '`+localTZ+`')::date = day),
(SELECT count(*) FROM participations p WHERE (p.created_at AT TIME ZONE '`+localTZ+`')::date = day)
FROM generate_series((now() AT TIME ZONE '`+localTZ+`')::date - 13, (now() AT TIME ZONE '`+localTZ+`')::date, interval '1 day') day`)
	if err != nil {
		return d, err
	}
	if d.Series, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (DayStat, error) {
		var s DayStat
		return s, row.Scan(&s.Date, &s.Events, &s.Participations)
	}); err != nil {
		return d, err
	}
	rows, err = tx.Query(ctx, `SELECT sport, count(*) FROM events WHERE status='published' AND starts_at > now()
GROUP BY sport ORDER BY count(*) DESC, sport LIMIT 6`)
	if err != nil {
		return d, err
	}
	if d.TopSports, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (SportStat, error) {
		var s SportStat
		return s, row.Scan(&s.Sport, &s.Events)
	}); err != nil {
		return d, err
	}
	rows, err = tx.Query(ctx, `SELECT id::text, COALESCE(name,''), COALESCE(owner_name,''), created_at FROM organizations
WHERE status='pending' ORDER BY created_at, id LIMIT 5`)
	if err != nil {
		return d, err
	}
	if d.PendingOrganizations, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (PendingOrganization, error) {
		var o PendingOrganization
		return o, row.Scan(&o.ID, &o.Name, &o.OwnerName, &o.CreatedAt)
	}); err != nil {
		return d, err
	}
	return d, tx.Commit(ctx)
}

// LogEntry — решение в журнале модерации.
type LogEntry struct {
	Entity   string    `json:"entity"`
	EntityID string    `json:"entity_id"`
	Title    string    `json:"title"`
	From     string    `json:"from"`
	To       string    `json:"to"`
	Actor    string    `json:"actor"`
	Comment  string    `json:"comment"`
	At       time.Time `json:"at"`
}

// Comment — сообщение в ленте модерации: последние сообщения всех обсуждений в одном месте.
type Comment struct {
	ID           string    `json:"id"`
	EventID      string    `json:"event_id"`
	EventTitle   string    `json:"event_title"`
	AuthorName   string    `json:"author_name"`
	Body         string    `json:"body"`
	Status       string    `json:"status"`
	HiddenReason string    `json:"hidden_reason"`
	CreatedAt    time.Time `json:"created_at"`
}

type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

var logSpec = platform.ListSpec{
	Filters:     map[string]string{"entity": "entity", "to": "to_status", "entity_id": "entity_id::text"},
	Sorts:       map[string]string{"at": "at"},
	DefaultSort: "-at",
	Search:      []string{"title", "actor", "comment"},
}

var commentSpec = platform.ListSpec{
	Filters:     map[string]string{"status": "status", "event_id": "event_id::text"},
	Sorts:       map[string]string{"created": "created_at"},
	DefaultSort: "-created",
	Search:      []string{"body", "author_name", "event_title"},
}

// page — общая выборка страницы по ListQuery: count и строки в одном снимке.
func page[T any](ctx context.Context, r *Repository, from, columns string, q platform.ListQuery, scan func(pgx.CollectableRow) (T, error)) (Page[T], error) {
	out := Page[T]{Items: []T{}, Limit: q.Limit, Offset: q.Offset}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	where := " WHERE true" + q.Where
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM "+from+where, q.Args...).Scan(&out.Total); err != nil {
		return out, err
	}
	n := len(q.Args)
	rows, err := tx.Query(ctx, "SELECT "+columns+" FROM "+from+where+q.OrderBy+
		" LIMIT $"+strconv.Itoa(n+1)+" OFFSET $"+strconv.Itoa(n+2), append(q.Args, q.Limit, q.Offset)...)
	if err != nil {
		return out, err
	}
	if out.Items, err = pgx.CollectRows(rows, scan); err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

const logColumns = "entity, entity_id::text, title, from_status, to_status, actor, comment, at"

func scanLog(row pgx.CollectableRow) (LogEntry, error) {
	var e LogEntry
	return e, row.Scan(&e.Entity, &e.EntityID, &e.Title, &e.From, &e.To, &e.Actor, &e.Comment, &e.At)
}

func (r *Repository) Log(ctx context.Context, q platform.ListQuery) (Page[LogEntry], error) {
	return page(ctx, r, "moderation_log", logColumns, q, scanLog)
}

func (r *Repository) Comments(ctx context.Context, q platform.ListQuery) (Page[Comment], error) {
	return page(ctx, r, "comments", `id::text, COALESCE(event_id::text,''), COALESCE(event_title,''), COALESCE(author_name,''),
COALESCE(body,''), COALESCE(status,''), COALESCE(hidden_reason,''), created_at`, q,
		func(row pgx.CollectableRow) (Comment, error) {
			var c Comment
			return c, row.Scan(&c.ID, &c.EventID, &c.EventTitle, &c.AuthorName, &c.Body, &c.Status, &c.HiddenReason, &c.CreatedAt)
		})
}

// Строки отчётов: колонки уже в том порядке, в каком их увидит человек в Excel.

func (r *Repository) EventRows(ctx context.Context) ([][]string, error) {
	return r.rows(ctx, `SELECT COALESCE(e.title,''), COALESCE(e.sport,''), COALESCE(e.organizer_name,''), COALESCE(e.organization_name,''),
COALESCE(to_char(e.starts_at AT TIME ZONE '`+localTZ+`', 'DD.MM.YYYY HH24:MI'),''), COALESCE(e.location,''),
CASE e.join_mode WHEN 'open' THEN 'свободная' WHEN 'request' THEN 'по заявке' ELSE '' END,
COALESCE(e.status,''), COALESCE(e.moderation_reason,''),
(SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='confirmed')::text,
(SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='pending')::text,
(SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='confirmed' AND p.first_time)::text,
COALESCE(e.capacity::text,''), COALESCE(e.min_participants::text,''), COALESCE(e.quorum_status,'')
FROM events e ORDER BY e.starts_at DESC NULLS LAST, e.id`)
}

func (r *Repository) OrganizationRows(ctx context.Context) ([][]string, error) {
	return r.rows(ctx, `SELECT COALESCE(o.name,''), COALESCE(o.city,''), COALESCE(o.owner_name,''), COALESCE(o.status,''),
COALESCE(o.review_reason,''), COALESCE(to_char(o.created_at AT TIME ZONE '`+localTZ+`', 'DD.MM.YYYY HH24:MI'),''),
(SELECT count(*) FROM events e WHERE e.organization_id=o.id)::text
FROM organizations o ORDER BY o.created_at, o.id`)
}

func (r *Repository) LogRows(ctx context.Context) ([][]string, error) {
	return r.rows(ctx, `SELECT to_char(at AT TIME ZONE '`+localTZ+`', 'DD.MM.YYYY HH24:MI'), entity, title, from_status, to_status, actor, comment
FROM moderation_log ORDER BY at DESC LIMIT 10000`)
}

// ReportRows — жалобы: открытые сначала, затем свежие. «Событие» пусто у истории и личного сообщения.
func (r *Repository) ReportRows(ctx context.Context) ([][]string, error) {
	return r.rows(ctx, `SELECT COALESCE(to_char(created_at AT TIME ZONE '`+localTZ+`', 'DD.MM.YYYY HH24:MI'),''), COALESCE(target_type,''),
COALESCE(event_title,''), COALESCE(reason,''), COALESCE(details,''), COALESCE(status,''), COALESCE(resolution,''), COALESCE(resolved_by,'')
FROM reports ORDER BY (status<>'open'), created_at DESC NULLS LAST, id LIMIT 10000`)
}

func (r *Repository) rows(ctx context.Context, sql string) ([][]string, error) {
	rows, err := r.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) ([]string, error) {
		values, err := row.Values()
		if err != nil {
			return nil, err
		}
		out := make([]string, len(values))
		for i, v := range values {
			out[i], _ = v.(string)
		}
		return out, nil
	})
}
