package tasks

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	rows, err := tx.Query(ctx, "SELECT id,title,completed,created_at FROM tasks WHERE owner_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3", owner, limit, offset)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt); err != nil {
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
func (r *Repository) Insert(ctx context.Context, owner string, t Task) error {
	_, err := r.db.Exec(ctx, "INSERT INTO tasks(id,owner_id,title,completed,created_at) VALUES($1,$2,$3,$4,$5)", t.ID, owner, t.Title, t.Completed, t.CreatedAt)
	return err
}
func (r *Repository) Update(ctx context.Context, owner, id string, completed bool) (Task, error) {
	var t Task
	err := r.db.QueryRow(ctx, "UPDATE tasks SET completed=$3 WHERE id=$1 AND owner_id=$2 RETURNING id,title,completed,created_at", id, owner, completed).Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return t, err
}
func (r *Repository) Delete(ctx context.Context, owner, id string) error {
	result, err := r.db.Exec(ctx, "DELETE FROM tasks WHERE id=$1 AND owner_id=$2", id, owner)
	if err == nil && result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
