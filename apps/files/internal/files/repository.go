package files

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository — реализация Store на Postgres.
type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, owner string, entity *Entity, limit, offset int) (Page, error) {
	where, args := "owner_id=$1", []any{owner}
	if entity != nil {
		where, args = "owner_id=$1 AND entity_type=$2 AND entity_id=$3", []any{owner, entity.Type, entity.ID}
	}
	page := Page{Items: []File{}, Limit: limit, Offset: offset}
	// Список и total читаются из одного снимка, даже если параллельно добавляют или удаляют файлы.
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM files WHERE "+where, args...).Scan(&page.Total); err != nil {
		return page, err
	}
	n := len(args)
	rows, err := tx.Query(ctx, "SELECT id,name,size,content_type,created_at,entity_type,entity_id FROM files WHERE "+where+
		" ORDER BY created_at DESC,id DESC LIMIT $"+strconv.Itoa(n+1)+" OFFSET $"+strconv.Itoa(n+2), append(args, limit, offset)...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var f File
		var entityType, entityID *string
		if err := rows.Scan(&f.ID, &f.Name, &f.Size, &f.ContentType, &f.CreatedAt, &entityType, &entityID); err != nil {
			return page, err
		}
		if entityType != nil && entityID != nil {
			f.Entity = &Entity{Type: *entityType, ID: *entityID}
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
	var entityType, entityID *string
	if f.Entity != nil {
		entityType, entityID = &f.Entity.Type, &f.Entity.ID
	}
	_, err := r.db.Exec(ctx, "INSERT INTO files(id,owner_id,object_key,name,size,content_type,created_at,entity_type,entity_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)",
		f.ID, owner, objectKey, f.Name, f.Size, f.ContentType, f.CreatedAt, entityType, entityID)
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
