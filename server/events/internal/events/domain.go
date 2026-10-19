package events

import (
	"context"
	"fmt"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Доменные события для других сервисов. Каждый переход статуса (record) в той же транзакции пишет
// историю и ставит в outbox событие «domain.<сущность>.changed» со снимком сущности после изменения.
// Подписчик (сервис moderation) строит из них свои таблицы: дашборд, журнал, отчёты — и не читает
// чужую базу. Снимок целиком, а не только переход: обработчик идемпотентен и не зависит от порядка.

// snapshots — состояние сущности после перехода; $1 — её id. Имена людей и названия берутся сразу,
// чтобы подписчику не нужно было ходить в этот сервис за подробностями.
var snapshots = map[string]string{
	"event": `SELECT to_jsonb(x) FROM (SELECT e.id, e.title, s.name AS sport, e.organizer_id,
  COALESCE(pr.display_name,'') AS organizer_name, e.organization_id, COALESCE(o.name,'') AS organization_name,
  e.starts_at, e.location, e.join_mode, e.capacity, e.min_participants, e.quorum_status, e.outdoor,
  e.status, e.moderation_reason, e.created_at
FROM events e JOIN sports s ON s.id=e.sport_id LEFT JOIN profiles pr ON pr.user_id=e.organizer_id
LEFT JOIN organizations o ON o.id=e.organization_id WHERE e.id=$1::uuid) x`,
	"organization": `SELECT to_jsonb(x) FROM (SELECT o.id, o.name, o.city, o.owner_id, COALESCE(pr.display_name,'') AS owner_name,
  o.status, o.review_reason, o.created_at
FROM organizations o LEFT JOIN profiles pr ON pr.user_id=o.owner_id WHERE o.id=$1::uuid) x`,
	"participation": `SELECT to_jsonb(x) FROM (SELECT p.id, p.event_id, e.title AS event_title, p.user_id,
  COALESCE(pr.display_name,'') AS user_name, p.status, p.first_time, p.created_at
FROM participations p JOIN events e ON e.id=p.event_id LEFT JOIN profiles pr ON pr.user_id=p.user_id WHERE p.id=$1::uuid) x`,
	"comment": `SELECT to_jsonb(x) FROM (SELECT c.id, c.event_id, e.title AS event_title, c.author_id,
  COALESCE(pr.display_name,'') AS author_name, left(c.body, 500) AS body, c.status, c.hidden_reason, c.created_at
FROM comments c JOIN events e ON e.id=c.event_id LEFT JOIN profiles pr ON pr.user_id=c.author_id WHERE c.id=$1::uuid) x`,
	"company_request": `SELECT to_jsonb(x) FROM (SELECT r.id, r.user_id, s.name AS sport, r.level, r.status, r.created_at
FROM company_requests r JOIN sports s ON s.id=r.sport_id WHERE r.id=$1::uuid) x`,
}

// entityOf — какая сущность описывает переход: смена состава — это изменение события.
var entityOf = map[string]string{"event_quorum": "event"}

// record — переход статуса: история + доменное событие в одной транзакции.
func record(ctx context.Context, tx pgx.Tx, entityType, id string, t platform.Transition) error {
	if err := platform.RecordTransition(ctx, tx, entityType, id, t); err != nil {
		return err
	}
	return emitDomain(ctx, tx, entityType, id, t)
}

func emitDomain(ctx context.Context, tx pgx.Tx, entityType, id string, t platform.Transition) error {
	entity := entityType
	if e, ok := entityOf[entityType]; ok {
		entity = e
	}
	snapshot, ok := snapshots[entity]
	if !ok {
		return fmt.Errorf("нет снимка для сущности %q", entity)
	}
	_, err := tx.Exec(ctx, `INSERT INTO outbox(id,type,user_id,request_id,payload)
SELECT $2, $3, '', $4, jsonb_build_object('entity',$5::text,'id',$1::text,'kind',$6::text,'from',$7::text,'to',$8::text,
  'actor',$9::text,'comment',$10::text,'at',now(),'snapshot',(`+snapshot+`))`,
		id, uuid.NewString(), "domain."+entity+".changed", platform.RequestID(ctx), entity, entityType, t.From, t.To, t.Actor, t.Comment)
	return err
}

// RepublishDomain ставит в outbox снимки всех сущностей — чтобы подписчик, подключённый позже
// (или после очистки своей базы), догнал текущее состояние. Подписчик идемпотентен, повтор безопасен.
func (r *Repository) RepublishDomain(ctx context.Context) (int, error) {
	total := 0
	for _, q := range []struct{ entity, ids string }{
		{"organization", "SELECT id::text FROM organizations"},
		{"event", "SELECT id::text FROM events"},
		{"participation", "SELECT id::text FROM participations"},
		{"comment", "SELECT id::text FROM comments"},
		{"company_request", "SELECT id::text FROM company_requests"},
	} {
		err := r.inTx(ctx, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, q.ids)
			if err != nil {
				return err
			}
			ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
			if err != nil {
				return err
			}
			for _, id := range ids {
				if err = emitDomain(ctx, tx, q.entity, id, platform.Transition{Actor: "snapshot"}); err != nil {
					return err
				}
			}
			total += len(ids)
			return nil
		})
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
