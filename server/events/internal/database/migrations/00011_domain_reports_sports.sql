-- +goose Up
-- Жалобы и направления тоже публикуются в moderation (журнал, дашборд, CSV): нужен счётчик версии снимка.
ALTER TABLE reports ADD COLUMN domain_version bigint NOT NULL DEFAULT 0;
ALTER TABLE sports ADD COLUMN domain_version bigint NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE sports DROP COLUMN domain_version;
ALTER TABLE reports DROP COLUMN domain_version;
