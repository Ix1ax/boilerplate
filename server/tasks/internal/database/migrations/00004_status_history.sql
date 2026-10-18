-- +goose Up
-- Универсальная история статусов: подходит для любых сущностей сервиса (entity_type + entity_id).
CREATE TABLE status_history (
  id bigserial PRIMARY KEY,
  entity_type text NOT NULL,
  entity_id text NOT NULL,
  from_status text NOT NULL,
  to_status text NOT NULL,
  actor text NOT NULL,
  comment text NOT NULL DEFAULT '' CHECK (char_length(comment) <= 2000),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX status_history_entity_idx ON status_history (entity_type, entity_id, created_at);
-- +goose Down
DROP TABLE status_history;
