package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"example.com/hackathon/apps/tasks/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }
func (r *Repository) List(ctx context.Context, owner string, limit, offset int) (Page, error) {
	page := Page{Items: []Task{}, Limit: limit, Offset: offset}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM tasks WHERE owner_id=$1", owner).Scan(&page.Total); err != nil {
		return page, err
	}
	rows, err := tx.Query(ctx, selectTask+" WHERE t.owner_id=$1 ORDER BY t.created_at DESC,t.id DESC LIMIT $2 OFFSET $3", owner, limit, offset)
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
	if _, err = tx.Exec(ctx, "INSERT INTO tasks(id,owner_id,title,completed,created_at) VALUES($1,$2,$3,$4,$5)", t.ID, owner, t.Title, t.Completed, t.CreatedAt); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"task_id": t.ID, "email": platform.Email(ctx)})
	if _, err = tx.Exec(ctx, "INSERT INTO outbox(id,type,user_id,request_id,payload) VALUES($1,'task.created',$2,$3,$4)", uuid.NewString(), owner, platform.RequestID(ctx), payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) Update(ctx context.Context, owner, id string, completed bool) (Task, error) {
	tag, err := r.db.Exec(ctx, "UPDATE tasks SET completed=$3 WHERE id=$1 AND owner_id=$2", id, owner, completed)
	if err != nil {
		return Task{}, err
	}
	if tag.RowsAffected() == 0 {
		return Task{}, ErrNotFound
	}
	return r.Get(ctx, owner, id)
}
func (r *Repository) Delete(ctx context.Context, owner, id string) error {
	result, err := r.db.Exec(ctx, "DELETE FROM tasks WHERE id=$1 AND owner_id=$2", id, owner)
	if err == nil && result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

const selectTask = `SELECT t.id,t.title,t.completed,t.created_at,m.verdict,m.source,m.checked_at
FROM tasks t LEFT JOIN task_moderation m ON m.task_id=t.id`

func scanTask(row pgx.Row) (Task, error) {
	var t Task
	var verdict, source *string
	var checked *time.Time
	if err := row.Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt, &verdict, &source, &checked); err != nil {
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
	rows, err := r.db.Query(ctx, `SELECT t.owner_id,t.id,t.title,t.completed,t.created_at,m.verdict,m.source,m.checked_at
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
		if err = rows.Scan(&owner, &t.ID, &t.Title, &t.Completed, &t.CreatedAt, &verdict, &source, &checked); err != nil {
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
func (r *Repository) Stats(ctx context.Context) (Stats, error) {
	st := Stats{ByDay: []DayCount{}}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return st, err
	}
	defer tx.Rollback(ctx)
	if err = tx.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE t.completed), count(DISTINCT t.owner_id),
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
