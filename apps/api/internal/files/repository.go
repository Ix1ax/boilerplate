package files

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository — реализация Store на Postgres.
type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, owner string, limit, offset int) (Page, error) {
	page := Page{Items: []File{}, Limit: limit, Offset: offset}
	// Список и total читаются из одного снимка, даже если параллельно добавляют или удаляют файлы.
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM files WHERE owner_id=$1", owner).Scan(&page.Total); err != nil {
		return page, err
	}
	rows, err := tx.Query(ctx, "SELECT id,name,size,content_type,created_at FROM files WHERE owner_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3", owner, limit, offset)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Name, &f.Size, &f.ContentType, &f.CreatedAt); err != nil {
			return page, err
		}
		page.Items = append(page.Items, f)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	rows.Close()
	return page, tx.Commit(ctx)
}

func (r *Repository) Insert(ctx context.Context, owner, objectKey string, f File) error {
	_, err := r.db.Exec(ctx, "INSERT INTO files(id,owner_id,object_key,name,size,content_type,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)",
		f.ID, owner, objectKey, f.Name, f.Size, f.ContentType, f.CreatedAt)
	return err
}

func (r *Repository) Get(ctx context.Context, owner, id string) (objectKey, name, contentType string, err error) {
	err = r.db.QueryRow(ctx, "SELECT object_key,name,content_type FROM files WHERE id=$1 AND owner_id=$2", id, owner).
		Scan(&objectKey, &name, &contentType)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return
}

// Delete допускает повтор после удаления объекта из S3, в том числе при параллельных запросах.
func (r *Repository) Delete(ctx context.Context, owner, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM files WHERE id=$1 AND owner_id=$2", id, owner)
	return err
}
