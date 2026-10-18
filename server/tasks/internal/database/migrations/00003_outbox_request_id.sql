-- +goose Up
-- Идентификатор запроса, породившего событие: по нему цепочка видна в логах всех сервисов.
ALTER TABLE outbox ADD COLUMN request_id text NOT NULL DEFAULT '';
-- +goose Down
ALTER TABLE outbox DROP COLUMN request_id;
