-- +goose Up
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
