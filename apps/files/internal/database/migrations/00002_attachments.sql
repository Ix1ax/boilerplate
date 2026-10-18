-- +goose Up
-- Привязка файла к записи другого сервиса (документы заявки и т.п.). Обе колонки пусты или обе заполнены.
ALTER TABLE files
  ADD COLUMN entity_type text,
  ADD COLUMN entity_id text,
  ADD CONSTRAINT files_entity_pair CHECK ((entity_type IS NULL) = (entity_id IS NULL));
CREATE INDEX files_owner_entity_idx ON files (owner_id, entity_type, entity_id, created_at DESC);
-- +goose Down
DROP INDEX files_owner_entity_idx;
ALTER TABLE files DROP CONSTRAINT files_entity_pair, DROP COLUMN entity_id, DROP COLUMN entity_type;
