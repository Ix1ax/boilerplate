package platform

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB — общее у *pgxpool.Pool и pgx.Tx: историю пишем в той же транзакции, что и смену статуса.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Transition — запись истории статусов любой сущности (таблица status_history).
type Transition struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Actor     string    `json:"actor"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// RecordTransition сохраняет переход. Вызывайте после Workflow.Check и в одной транзакции с UPDATE статуса.
func RecordTransition(ctx context.Context, db DB, entityType, entityID string, t Transition) error {
	_, err := db.Exec(ctx, `INSERT INTO status_history(entity_type,entity_id,from_status,to_status,actor,comment)
VALUES($1,$2,$3,$4,$5,$6)`, entityType, entityID, t.From, t.To, t.Actor, t.Comment)
	return err
}

// History — переходы сущности от старых к новым (лента «отправлено → на модерации → возвращено»).
func History(ctx context.Context, db DB, entityType, entityID string) ([]Transition, error) {
	rows, err := db.Query(ctx, `SELECT from_status,to_status,actor,comment,created_at FROM status_history
WHERE entity_type=$1 AND entity_id=$2 ORDER BY created_at, id`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Transition{}
	for rows.Next() {
		var t Transition
		if err = rows.Scan(&t.From, &t.To, &t.Actor, &t.Comment, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
