-- +goose Up
-- Сервис moderation — админка-аналитика. Свои таблицы строятся из доменных событий сервиса events
-- (domain.*.changed): это проекции для чтения, источник правды остаётся в events.
-- Внешних ключей нет намеренно: события приходят в любом порядке (запись раньше события).
-- changed_at — время изменения в events: более старый снимок не затирает новый.

CREATE TABLE organizations (
  id uuid PRIMARY KEY,
  name text,
  city text,
  owner_id text,
  owner_name text,
  status text,
  review_reason text,
  created_at timestamptz,
  changed_at timestamptz NOT NULL
);
CREATE INDEX organizations_status_idx ON organizations (status, created_at);

CREATE TABLE events (
  id uuid PRIMARY KEY,
  title text,
  sport text,
  organizer_id text,
  organizer_name text,
  organization_id uuid,
  organization_name text,
  starts_at timestamptz,
  location text,
  join_mode text,
  capacity int,
  min_participants int,
  quorum_status text,
  outdoor boolean,
  status text,
  moderation_reason text,
  created_at timestamptz,
  changed_at timestamptz NOT NULL
);
CREATE INDEX events_status_idx ON events (status, starts_at);
CREATE INDEX events_created_idx ON events (created_at);

CREATE TABLE participations (
  id uuid PRIMARY KEY,
  event_id uuid,
  event_title text,
  user_id text,
  user_name text,
  status text,
  first_time boolean,
  created_at timestamptz,
  changed_at timestamptz NOT NULL
);
CREATE INDEX participations_event_idx ON participations (event_id, status);

CREATE TABLE comments (
  id uuid PRIMARY KEY,
  event_id uuid,
  event_title text,
  author_id text,
  author_name text,
  body text,
  status text,
  hidden_reason text,
  created_at timestamptz,
  changed_at timestamptz NOT NULL
);
CREATE INDEX comments_created_idx ON comments (created_at DESC);

CREATE TABLE company_requests (
  id uuid PRIMARY KEY,
  user_id text,
  sport text,
  level text,
  status text,
  created_at timestamptz,
  changed_at timestamptz NOT NULL
);

-- Журнал модерации: решения по организациям, событиям и сообщениям, включая автоматические
-- («Платформа» отменила событие, не собравшее состав). message_id — id доменного события: повтор не дублирует.
CREATE TABLE moderation_log (
  message_id uuid PRIMARY KEY,
  entity text NOT NULL,
  entity_id uuid NOT NULL,
  kind text NOT NULL,
  title text NOT NULL DEFAULT '',
  from_status text NOT NULL,
  to_status text NOT NULL,
  actor text NOT NULL,
  comment text NOT NULL DEFAULT '',
  at timestamptz NOT NULL
);
CREATE INDEX moderation_log_at_idx ON moderation_log (at DESC);
CREATE INDEX moderation_log_entity_idx ON moderation_log (entity, entity_id, at);

-- +goose Down
DROP TABLE moderation_log;
DROP TABLE company_requests;
DROP TABLE comments;
DROP TABLE participations;
DROP TABLE events;
DROP TABLE organizations;
