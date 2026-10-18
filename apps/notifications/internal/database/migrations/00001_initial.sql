-- +goose Up
CREATE TABLE notifications (
  id uuid PRIMARY KEY,
  user_id text NOT NULL,
  title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
  body text NOT NULL DEFAULT '' CHECK (char_length(body) <= 1000),
  link text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  read_at timestamptz,
  -- Идемпотентность: повторная доставка того же события не создаёт дубль.
  source_event_id uuid UNIQUE
);
CREATE INDEX notifications_user_created_idx ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX notifications_unread_idx ON notifications (user_id) WHERE read_at IS NULL;
-- +goose Down
DROP TABLE notifications;
