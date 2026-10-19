-- +goose Up
-- Жалобы и справочник направлений из domain.report.changed / domain.sport.changed: очередь жалоб
-- на дашборде, их выгрузка и пользовательские направления. Колонки совпадают с ключами снимка.
-- event_id пустой у жалоб на историю и личное сообщение: у них нет мероприятия.
CREATE TABLE reports (
  id uuid PRIMARY KEY,
  target_type text,
  target_id uuid,
  event_id uuid,
  event_title text,
  target_title text,
  reason text,
  details text,
  status text,
  resolution text,
  resolved_by text,
  reporter_id text,
  reporter_name text,
  created_at timestamptz,
  resolved_at timestamptz,
  changed_at timestamptz NOT NULL,
  domain_version bigint NOT NULL DEFAULT 0
);
CREATE INDEX reports_status_idx ON reports (status, resolved_at);
CREATE INDEX reports_created_idx ON reports (created_at DESC);

CREATE TABLE sports (
  id uuid PRIMARY KEY,
  name text,
  is_custom boolean,
  hidden boolean,
  created_by text,
  created_by_name text,
  created_at timestamptz,
  changed_at timestamptz NOT NULL,
  domain_version bigint NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE sports;
DROP TABLE reports;
