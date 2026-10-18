-- +goose Up
-- Конечный автомат статусов вместо булева completed: нужен для цветных статусов,
-- ленты истории переходов (status_history) и фильтра по статусу в списке задач.
ALTER TABLE tasks ADD COLUMN status text NOT NULL DEFAULT 'todo'
  CHECK (status IN ('draft','todo','in_progress','done','archived'));
UPDATE tasks SET status = CASE WHEN completed THEN 'done' ELSE 'todo' END;
ALTER TABLE tasks DROP COLUMN completed;
CREATE INDEX tasks_status_idx ON tasks (owner_id, status);
-- +goose Down
ALTER TABLE tasks ADD COLUMN completed boolean NOT NULL DEFAULT false;
UPDATE tasks SET completed = (status = 'done');
DROP INDEX tasks_status_idx;
ALTER TABLE tasks DROP COLUMN status;
