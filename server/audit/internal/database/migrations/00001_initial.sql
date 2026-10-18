-- +goose Up
-- Журнал изменяющих запросов: кто, что и когда сделал. Записи только добавляются.
CREATE TABLE audit_log (
  id bigserial PRIMARY KEY,
  event_id uuid UNIQUE NOT NULL,
  actor text NOT NULL,
  actor_email text NOT NULL DEFAULT '',
  method text NOT NULL,
  route text NOT NULL,
  path text NOT NULL,
  status int NOT NULL,
  request_id text NOT NULL DEFAULT '',
  ip text NOT NULL DEFAULT '',
  occurred_at timestamptz NOT NULL
);
CREATE INDEX audit_log_occurred_idx ON audit_log (occurred_at DESC);
CREATE INDEX audit_log_actor_idx ON audit_log (actor, occurred_at DESC);
-- +goose Down
DROP TABLE audit_log;
