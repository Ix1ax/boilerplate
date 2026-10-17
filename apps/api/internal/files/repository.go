package files

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Page struct {
	Items  []File `json:"items"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

func List(ctx context.Context, db *pgxpool.Pool, owner string, limit, offset int) (Page, error) {
	page := Page{Items: []File{}, Limit: limit, Offset: offset}
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
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
