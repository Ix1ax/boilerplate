-- +goose Up
-- Версия снимка растёт под блокировкой сущности; порядок начала транзакций не важен.
ALTER TABLE organizations ADD COLUMN domain_version bigint NOT NULL DEFAULT 0;
ALTER TABLE events ADD COLUMN domain_version bigint NOT NULL DEFAULT 0;
ALTER TABLE participations ADD COLUMN domain_version bigint NOT NULL DEFAULT 0;
ALTER TABLE comments ADD COLUMN domain_version bigint NOT NULL DEFAULT 0;
ALTER TABLE company_requests ADD COLUMN domain_version bigint NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE organizations DROP COLUMN domain_version;
ALTER TABLE events DROP COLUMN domain_version;
ALTER TABLE participations DROP COLUMN domain_version;
ALTER TABLE comments DROP COLUMN domain_version;
ALTER TABLE company_requests DROP COLUMN domain_version;
