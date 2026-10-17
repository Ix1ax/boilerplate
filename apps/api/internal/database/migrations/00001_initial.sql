-- +goose Up
CREATE TABLE tasks (
  id uuid PRIMARY KEY,
  owner_id text NOT NULL,
  title text NOT NULL CHECK (char_length(btrim(title)) BETWEEN 1 AND 200),
  completed boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tasks_owner_created_idx ON tasks (owner_id, created_at DESC, id DESC);
CREATE TABLE files (
  id uuid PRIMARY KEY,
  owner_id text NOT NULL,
  object_key text NOT NULL UNIQUE,
  name text NOT NULL,
  size bigint NOT NULL CHECK (size BETWEEN 1 AND 10485760),
  content_type text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX files_owner_idx ON files(owner_id, created_at DESC);
-- +goose Down
DROP TABLE files;
DROP TABLE tasks;
