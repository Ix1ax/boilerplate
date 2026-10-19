-- +goose Up
ALTER TABLE events ADD COLUMN format text NOT NULL DEFAULT 'offline'
  CHECK (format IN ('offline', 'online'));

-- +goose Down
ALTER TABLE events DROP COLUMN format;
