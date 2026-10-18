package audit

import (
	"context"

	"example.com/hackathon/apps/audit/internal/platform"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Insert(ctx context.Context, eventID string, rec Record) error {
	_, err := r.db.Exec(ctx, `INSERT INTO audit_log(event_id,actor,actor_email,method,route,path,status,request_id,ip,occurred_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (event_id) DO NOTHING`,
		eventID, rec.ActorID, rec.ActorEmail, rec.Method, rec.Route, rec.Path, rec.Status, rec.RequestID, rec.IP, rec.OccurredAt)
	return err
}

// List — страница журнала с фильтрами из platform.ParseList (колонки только из белого списка).
func (r *Repository) List(ctx context.Context, q platform.ListQuery) (Page, error) {
	page := Page{Items: []Record{}, Limit: q.Limit, Offset: q.Offset}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	// ParseList нумерует параметры с $1: условий владельца здесь нет, журнал общий для админа.
	where := "WHERE true" + q.Where
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM audit_log "+where, q.Args...).Scan(&page.Total); err != nil {
		return page, err
	}
	n := len(q.Args)
	rows, err := tx.Query(ctx, "SELECT actor,actor_email,method,route,path,status,request_id,ip,occurred_at FROM audit_log "+where+q.OrderBy+
		", id DESC LIMIT $"+itoa(n+1)+" OFFSET $"+itoa(n+2), append(q.Args, q.Limit, q.Offset)...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var rec Record
		if err = rows.Scan(&rec.ActorID, &rec.ActorEmail, &rec.Method, &rec.Route, &rec.Path, &rec.Status, &rec.RequestID, &rec.IP, &rec.OccurredAt); err != nil {
			return page, err
		}
		page.Items = append(page.Items, rec)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	rows.Close()
	return page, tx.Commit(ctx)
}
