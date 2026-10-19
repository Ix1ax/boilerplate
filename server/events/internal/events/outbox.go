package events

import (
	"context"

	"example.com/hackathon/server/events/internal/broker"
)

// ClaimEvents временно (на 30 с) закрепляет за вызывающим до n неотправленных событий.
// SKIP LOCKED позволяет нескольким репликам работать параллельно без дублей на одной выборке.
func (r *Repository) ClaimEvents(ctx context.Context, n int) ([]broker.Event, error) {
	rows, err := r.db.Query(ctx, `UPDATE outbox SET locked_until = now() + interval '30 seconds'
WHERE id IN (SELECT id FROM outbox WHERE sent_at IS NULL AND (locked_until IS NULL OR locked_until < now())
              ORDER BY created_at LIMIT $1 FOR UPDATE SKIP LOCKED)
RETURNING id, type, user_id, request_id, payload, created_at`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []broker.Event
	for rows.Next() {
		var e broker.Event
		if err = rows.Scan(&e.ID, &e.Type, &e.UserID, &e.RequestID, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// MarkSent отмечает событие отправленным.
func (r *Repository) MarkSent(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "UPDATE outbox SET sent_at = now() WHERE id=$1", id)
	return err
}

// CleanupOutbox удаляет отправленные события старше суток (фоновое задание, см. platform.Every).
func (r *Repository) CleanupOutbox(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "DELETE FROM outbox WHERE sent_at < now() - interval '1 day'")
	return err
}
