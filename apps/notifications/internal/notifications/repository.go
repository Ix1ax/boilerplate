package notifications

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, user string, limit, offset int) (Page, error) {
	page := Page{Items: []Notification{}, Limit: limit, Offset: offset}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	if err = tx.QueryRow(ctx, "SELECT count(*), count(*) FILTER (WHERE read_at IS NULL) FROM notifications WHERE user_id=$1", user).Scan(&page.Total, &page.Unread); err != nil {
		return page, err
	}
	rows, err := tx.Query(ctx, "SELECT id,title,body,link,created_at,read_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3", user, limit, offset)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var n Notification
		if err = rows.Scan(&n.ID, &n.Title, &n.Body, &n.Link, &n.CreatedAt, &n.ReadAt); err != nil {
			return page, err
		}
		page.Items = append(page.Items, n)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	rows.Close()
	return page, tx.Commit(ctx)
}

// Insert идемпотентен по source_event_id: повторная доставка события ничего не меняет.
func (r *Repository) Insert(ctx context.Context, user, eventID string, n Notification) (bool, error) {
	tag, err := r.db.Exec(ctx, "INSERT INTO notifications(id,user_id,title,body,link,created_at,source_event_id) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (source_event_id) DO NOTHING", n.ID, user, n.Title, n.Body, n.Link, n.CreatedAt, eventID)
	return err == nil && tag.RowsAffected() == 1, err
}
func (r *Repository) MarkRead(ctx context.Context, user, id string) error {
	tag, err := r.db.Exec(ctx, "UPDATE notifications SET read_at=coalesce(read_at, now()) WHERE id=$1 AND user_id=$2", id, user)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (r *Repository) MarkAllRead(ctx context.Context, user string) error {
	_, err := r.db.Exec(ctx, "UPDATE notifications SET read_at=now() WHERE user_id=$1 AND read_at IS NULL", user)
	return err
}
