-- +goose Up
-- Transactional outbox: событие пишется в той же транзакции, что и данные, затем уходит в RabbitMQ.
CREATE TABLE outbox (
  id uuid PRIMARY KEY,
  type text NOT NULL,
  user_id text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  locked_until timestamptz,
  sent_at timestamptz
);
CREATE INDEX outbox_pending_idx ON outbox (created_at) WHERE sent_at IS NULL;
-- Результат асинхронной ИИ-проверки; удаляется вместе с задачей.
CREATE TABLE task_moderation (
  task_id uuid PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
  verdict text NOT NULL CHECK (verdict IN ('ok', 'needs_review')),
  source text NOT NULL,
  result jsonb NOT NULL,
  checked_at timestamptz NOT NULL DEFAULT now()
);
-- +goose Down
DROP TABLE task_moderation;
DROP TABLE outbox;
