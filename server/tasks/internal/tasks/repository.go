package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"example.com/hackathon/server/tasks/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
	"time"
)

// Repository — реализация Store на Postgres.
type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }
func (r *Repository) List(ctx context.Context, owner string, q platform.ListQuery) (Page, error) {
	page := Page{Items: []Task{}, Limit: q.Limit, Offset: q.Offset}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	where := "WHERE t.owner_id=$1" + q.Where
	args := append([]any{owner}, q.Args...)
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM tasks t "+where, args...).Scan(&page.Total); err != nil {
		return page, err
	}
	n := len(args)
	rows, err := tx.Query(ctx, selectTask+" "+where+q.OrderBy+", t.id DESC LIMIT $"+strconv.Itoa(n+1)+" OFFSET $"+strconv.Itoa(n+2),
		append(args, q.Limit, q.Offset)...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, t)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	rows.Close()
	return page, tx.Commit(ctx)
}
func (r *Repository) Get(ctx context.Context, owner, id string) (Task, error) {
	t, err := scanTask(r.db.QueryRow(ctx, selectTask+" WHERE t.id=$1 AND t.owner_id=$2", id, owner))
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	return t, err
}

// Insert сохраняет задачу и событие task.created одной транзакцией (outbox): даже если брокер
// недоступен, событие не потеряется и уйдёт позже.
func (r *Repository) Insert(ctx context.Context, owner string, t Task) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "INSERT INTO tasks(id,owner_id,title,status,created_at) VALUES($1,$2,$3,$4,$5)", t.ID, owner, t.Title, t.Status, t.CreatedAt); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"task_id": t.ID, "email": platform.Email(ctx)})
	if _, err = tx.Exec(ctx, "INSERT INTO outbox(id,type,user_id,request_id,payload) VALUES($1,'task.created',$2,$3,$4)", uuid.NewString(), owner, platform.RequestID(ctx), payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpdateStatus проверяет допустимость перехода (check) внутри транзакции с блокировкой строки,
// затем меняет статус и пишет историю одной транзакцией (platform.RecordTransition).
func (r *Repository) UpdateStatus(ctx context.Context, owner, id, to, comment, actor string, check func(from string) error) (Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback(ctx)
	var from string
	err = tx.QueryRow(ctx, "SELECT status FROM tasks WHERE id=$1 AND owner_id=$2 FOR UPDATE", id, owner).Scan(&from)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, err
	}
	if err = check(from); err != nil {
		return Task{}, err
	}
	if _, err = tx.Exec(ctx, "UPDATE tasks SET status=$3 WHERE id=$1 AND owner_id=$2", id, owner, to); err != nil {
		return Task{}, err
	}
	if err = platform.RecordTransition(ctx, tx, "task", id, platform.Transition{From: from, To: to, Actor: actor, Comment: comment}); err != nil {
		return Task{}, err
	}
	t, err := scanTask(tx.QueryRow(ctx, selectTask+" WHERE t.id=$1 AND t.owner_id=$2", id, owner))
	if err != nil {
		return Task{}, err
	}
	return t, tx.Commit(ctx)
}

// History — лента переходов статуса задачи (без проверки владения: см. Service.History).
func (r *Repository) History(ctx context.Context, id string) ([]platform.Transition, error) {
	return platform.History(ctx, r.db, "task", id)
}
func (r *Repository) Delete(ctx context.Context, owner, id string) error {
	result, err := r.db.Exec(ctx, "DELETE FROM tasks WHERE id=$1 AND owner_id=$2", id, owner)
	if err == nil && result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

const selectTask = `SELECT t.id,t.title,t.status,t.created_at,m.verdict,m.source,m.checked_at
FROM tasks t LEFT JOIN task_moderation m ON m.task_id=t.id`

func scanTask(row pgx.Row) (Task, error) {
	var t Task
	var verdict, source *string
	var checked *time.Time
	if err := row.Scan(&t.ID, &t.Title, &t.Status, &t.CreatedAt, &verdict, &source, &checked); err != nil {
		return Task{}, err
	}
	if verdict != nil && source != nil && checked != nil {
		t.Moderation = &Moderation{Verdict: *verdict, Source: *source, CheckedAt: *checked}
	}
	return t, nil
}

// ExportAll — админская выгрузка всех задач. Это осознанное исключение из правила «владелец в SQL»:
// маршрут доступен только роли admin (проверка в gateway и в сервисе).
func (r *Repository) ExportAll(ctx context.Context, row func(owner string, t Task) error) error {
	rows, err := r.db.Query(ctx, `SELECT t.owner_id,t.id,t.title,t.status,t.created_at,m.verdict,m.source,m.checked_at
FROM tasks t LEFT JOIN task_moderation m ON m.task_id=t.id ORDER BY t.created_at`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var owner string
		var t Task
		var verdict, source *string
		var checked *time.Time
		if err = rows.Scan(&owner, &t.ID, &t.Title, &t.Status, &t.CreatedAt, &verdict, &source, &checked); err != nil {
			return err
		}
		if verdict != nil && source != nil && checked != nil {
			t.Moderation = &Moderation{Verdict: *verdict, Source: *source, CheckedAt: *checked}
		}
		if err = row(owner, t); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Stats — сводка для админского дашборда: итоги, разбивка по проверке и динамика за 14 дней.
// Один снимок данных (RepeatableRead), чтобы цифры в карточках и на графике сходились.
// Поле Completed отражает количество задач со статусом done (имя сохранено для обратной совместимости контракта).
func (r *Repository) Stats(ctx context.Context) (Stats, error) {
	st := Stats{ByDay: []DayCount{}}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return st, err
	}
	defer tx.Rollback(ctx)
	if err = tx.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE t.status='done'), count(DISTINCT t.owner_id),
  count(*) FILTER (WHERE m.verdict='ok'), count(*) FILTER (WHERE m.verdict='needs_review'), count(*) FILTER (WHERE m.task_id IS NULL)
FROM tasks t LEFT JOIN task_moderation m ON m.task_id=t.id`).Scan(&st.Total, &st.Completed, &st.Owners, &st.Moderation.OK, &st.Moderation.NeedsReview, &st.Moderation.Unchecked); err != nil {
		return st, err
	}
	rows, err := tx.Query(ctx, `SELECT d::date, count(t.id) FROM generate_series(current_date - 13, current_date, interval '1 day') d
LEFT JOIN tasks t ON t.created_at >= d AND t.created_at < d + interval '1 day' GROUP BY d ORDER BY d`)
	if err != nil {
		return st, err
	}
	defer rows.Close()
	for rows.Next() {
		var day DayCount
		var date time.Time
		if err = rows.Scan(&date, &day.Count); err != nil {
			return st, err
		}
		day.Date = date.Format("2006-01-02")
		st.ByDay = append(st.ByDay, day)
	}
	if err = rows.Err(); err != nil {
		return st, err
	}
	rows.Close()
	return st, tx.Commit(ctx)
}
