package tasks

import (
	"context"
	"encoding/json"

	"example.com/hackathon/apps/tasks/internal/broker"
	"example.com/hackathon/apps/tasks/internal/platform"
	"github.com/google/uuid"
)

type Event = broker.Event

// ClaimEvents временно (на 30 с) закрепляет за вызывающим до n неотправленных событий.
// SKIP LOCKED позволяет нескольким репликам работать параллельно без дублей на одной выборке.
func (r *Repository) ClaimEvents(ctx context.Context, n int) ([]Event, error) {
	rows, err := r.db.Query(ctx, `UPDATE outbox SET locked_until = now() + interval '30 seconds'
WHERE id IN (SELECT id FROM outbox WHERE sent_at IS NULL AND (locked_until IS NULL OR locked_until < now())
              ORDER BY created_at LIMIT $1 FOR UPDATE SKIP LOCKED)
RETURNING id, type, user_id, request_id, payload, created_at`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
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

// SaveModeration записывает (или обновляет) результат проверки: операция идемпотентна.
// При первой проверке в той же транзакции ставится событие notification.requested для владельца —
// уведомления живут в отдельном сервисе и узнают о проверке только через брокер.
func (r *Repository) SaveModeration(ctx context.Context, owner, taskID string, m Moderation, result []byte) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var inserted bool
	if err = tx.QueryRow(ctx, `INSERT INTO task_moderation(task_id,verdict,source,result,checked_at) VALUES($1,$2,$3,$4,now())
ON CONFLICT (task_id) DO UPDATE SET verdict=EXCLUDED.verdict, source=EXCLUDED.source, result=EXCLUDED.result, checked_at=now()
RETURNING (xmax = 0)`, taskID, m.Verdict, m.Source, result).Scan(&inserted); err != nil {
		return err
	}
	if inserted {
		title := "ИИ-проверка: замечаний нет"
		if m.Verdict == "needs_review" {
			title = "ИИ-проверка: задача требует внимания"
		}
		// email непустой — notifications дополнительно отправит письмо (если настроен SMTP).
		payload, _ := json.Marshal(map[string]string{"title": title, "body": "Откройте задачу, чтобы посмотреть рекомендацию. Решение остаётся за вами.", "link": "/dashboard/tasks", "email": platform.Email(ctx)})
		if _, err = tx.Exec(ctx, "INSERT INTO outbox(id,type,user_id,request_id,payload) VALUES($1,'notification.requested',$2,$3,$4)", uuid.NewString(), owner, platform.RequestID(ctx), payload); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
