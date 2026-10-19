-- +goose Up
-- Домен кейса «Платформа совместных спортивных активностей» (docs/data-model.md).
-- Пользователи живут в Keycloak; здесь — только их публичный профиль, ключ — sub из токена.

CREATE TABLE profiles (
  user_id text PRIMARY KEY,
  display_name text NOT NULL CHECK (char_length(btrim(display_name)) BETWEEN 1 AND 100),
  about text NOT NULL DEFAULT '' CHECK (char_length(about) <= 1000),
  city text NOT NULL DEFAULT '' CHECK (char_length(city) <= 100),
  level text CHECK (level IN ('beginner', 'amateur', 'pro')),
  created_at timestamptz NOT NULL DEFAULT now()
);

-- Справочник направлений: базовые заводит команда, свои — пользователи при создании события.
-- Уникальность без учёта регистра и пробелов по краям: «Фрисби» и « фрисби » — одно направление.
CREATE TABLE sports (
  id uuid PRIMARY KEY,
  name text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 2 AND 60),
  is_custom boolean NOT NULL DEFAULT false,
  created_by text,
  hidden boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX sports_name_key ON sports (lower(btrim(name)));

CREATE TABLE organizations (
  id uuid PRIMARY KEY,
  -- Одна организация на представителя: несколько управляющих кейс не требует.
  owner_id text NOT NULL UNIQUE,
  name text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 2 AND 120),
  description text NOT NULL CHECK (char_length(description) <= 3000),
  city text NOT NULL DEFAULT '' CHECK (char_length(city) <= 100),
  contacts text NOT NULL DEFAULT '' CHECK (char_length(contacts) <= 500),
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
  -- Причина отказа видна представителю; без неё отклонить нельзя.
  review_reason text NOT NULL DEFAULT '' CHECK (char_length(review_reason) <= 1000),
  reviewed_by text,
  reviewed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (status <> 'rejected' OR btrim(review_reason) <> '')
);
CREATE INDEX organizations_status_idx ON organizations (status, created_at DESC);

CREATE TABLE organization_sports (
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  sport_id uuid NOT NULL REFERENCES sports(id),
  PRIMARY KEY (organization_id, sport_id)
);

CREATE TABLE events (
  id uuid PRIMARY KEY,
  -- Кто создал событие и управляет им (для события организации — её представитель).
  organizer_id text NOT NULL,
  -- NULL — событие от своего имени.
  organization_id uuid REFERENCES organizations(id),
  sport_id uuid NOT NULL REFERENCES sports(id),
  title text NOT NULL CHECK (char_length(btrim(title)) BETWEEN 3 AND 120),
  description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 5000),
  starts_at timestamptz NOT NULL,
  ends_at timestamptz CHECK (ends_at IS NULL OR ends_at > starts_at),
  location text NOT NULL CHECK (char_length(btrim(location)) BETWEEN 3 AND 300),
  lat double precision CHECK (lat BETWEEN -90 AND 90),
  lng double precision CHECK (lng BETWEEN -180 AND 180),
  level text NOT NULL DEFAULT 'any' CHECK (level IN ('any', 'beginner', 'amateur', 'pro')),
  requirements text NOT NULL DEFAULT '' CHECK (char_length(requirements) <= 2000),
  price_note text NOT NULL DEFAULT '' CHECK (char_length(price_note) <= 200),
  -- Точка сбора, контакты: только подтверждённым участникам и организатору.
  private_info text NOT NULL DEFAULT '' CHECK (char_length(private_info) <= 2000),
  join_mode text NOT NULL CHECK (join_mode IN ('open', 'request')),
  capacity int CHECK (capacity IS NULL OR capacity >= 1),
  status text NOT NULL DEFAULT 'published' CHECK (status IN ('published', 'unpublished', 'deleted', 'cancelled')),
  moderation_reason text NOT NULL DEFAULT '' CHECK (char_length(moderation_reason) <= 1000),
  moderated_by text,
  moderated_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  search tsvector GENERATED ALWAYS AS (
    to_tsvector('russian', title || ' ' || description || ' ' || location)
  ) STORED,
  -- Снятие и удаление админом — только с причиной: её видят организатор и участники.
  CHECK (status NOT IN ('unpublished', 'deleted') OR btrim(moderation_reason) <> '')
);
CREATE INDEX events_feed_idx ON events (status, starts_at);
CREATE INDEX events_sport_idx ON events (sport_id, starts_at);
CREATE INDEX events_organization_idx ON events (organization_id, starts_at);
CREATE INDEX events_organizer_idx ON events (organizer_id, starts_at);
CREATE INDEX events_search_idx ON events USING gin (search);

CREATE TABLE participations (
  id uuid PRIMARY KEY,
  event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  user_id text NOT NULL,
  status text NOT NULL CHECK (status IN ('confirmed', 'pending', 'rejected', 'cancelled', 'waitlist')),
  message text NOT NULL DEFAULT '' CHECK (char_length(message) <= 500),
  decision_reason text NOT NULL DEFAULT '' CHECK (char_length(decision_reason) <= 500),
  decided_by text,
  decided_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  -- Повторная запись обновляет ту же строку, дублей не бывает.
  UNIQUE (event_id, user_id)
);
CREATE INDEX participations_event_idx ON participations (event_id, status, created_at);
CREATE INDEX participations_user_idx ON participations (user_id, created_at DESC);

CREATE TABLE comments (
  id uuid PRIMARY KEY,
  event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  author_id text NOT NULL,
  -- Ответ на вопрос; один уровень вложенности проверяет сервис.
  parent_id uuid REFERENCES comments(id) ON DELETE CASCADE,
  body text NOT NULL CHECK (char_length(btrim(body)) BETWEEN 1 AND 2000),
  status text NOT NULL DEFAULT 'visible' CHECK (status IN ('visible', 'hidden')),
  hidden_reason text NOT NULL DEFAULT '' CHECK (char_length(hidden_reason) <= 500),
  hidden_by text,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (status <> 'hidden' OR btrim(hidden_reason) <> '')
);
CREATE INDEX comments_event_idx ON comments (event_id, created_at);

-- Transactional outbox: событие пишется в той же транзакции, что и данные, затем уходит в RabbitMQ.
CREATE TABLE outbox (
  id uuid PRIMARY KEY,
  type text NOT NULL,
  user_id text NOT NULL,
  request_id text NOT NULL DEFAULT '',
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  locked_until timestamptz,
  sent_at timestamptz
);
CREATE INDEX outbox_pending_idx ON outbox (created_at) WHERE sent_at IS NULL;

-- История статусов организаций, событий, записей и сообщений: кто, когда и почему.
CREATE TABLE status_history (
  id bigserial PRIMARY KEY,
  entity_type text NOT NULL,
  entity_id text NOT NULL,
  from_status text NOT NULL,
  to_status text NOT NULL,
  actor text NOT NULL,
  comment text NOT NULL DEFAULT '' CHECK (char_length(comment) <= 2000),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX status_history_entity_idx ON status_history (entity_type, entity_id, created_at);

-- Базовый справочник направлений. Свои пользователи добавляют без правки кода.
INSERT INTO sports (id, name) VALUES
  (gen_random_uuid(), 'Бег'),
  (gen_random_uuid(), 'Футбол'),
  (gen_random_uuid(), 'Баскетбол'),
  (gen_random_uuid(), 'Волейбол'),
  (gen_random_uuid(), 'Шахматы'),
  (gen_random_uuid(), 'Йога'),
  (gen_random_uuid(), 'Зарядка во дворе'),
  (gen_random_uuid(), 'Велоспорт'),
  (gen_random_uuid(), 'Плавание'),
  (gen_random_uuid(), 'Теннис'),
  (gen_random_uuid(), 'Настольный теннис'),
  (gen_random_uuid(), 'Скандинавская ходьба'),
  (gen_random_uuid(), 'Лыжи'),
  (gen_random_uuid(), 'Скалолазание');

-- +goose Down
DROP TABLE status_history;
DROP TABLE outbox;
DROP TABLE comments;
DROP TABLE participations;
DROP TABLE events;
DROP TABLE organization_sports;
DROP TABLE organizations;
DROP TABLE sports;
DROP TABLE profiles;
