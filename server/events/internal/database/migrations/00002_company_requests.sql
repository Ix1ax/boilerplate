-- +goose Up
-- «Ищу компанию»: человек описывает, чем хочет заниматься, если подходящего события пока нет.
-- Когда кто-то публикует подходящее событие, совпадение записывается в company_matches,
-- а человеку уходит уведомление. Организатор видит только обезличенный спрос (сколько ищут).

CREATE TABLE company_requests (
  id uuid PRIMARY KEY,
  user_id text NOT NULL,
  sport_id uuid NOT NULL REFERENCES sports(id),
  -- Уровень самого человека; any — не важно. Совместимость с уровнем события — в repository.go.
  level text NOT NULL DEFAULT 'any' CHECK (level IN ('any', 'beginner', 'amateur', 'pro')),
  -- Пустой массив — подходит любое время / любой день. Дни — ISO: 1 понедельник … 7 воскресенье.
  day_parts text[] NOT NULL DEFAULT '{}' CHECK (day_parts <@ ARRAY['morning', 'day', 'evening']),
  weekdays smallint[] NOT NULL DEFAULT '{}' CHECK (weekdays <@ ARRAY[1, 2, 3, 4, 5, 6, 7]::smallint[]),
  -- area — подпись для человека («рядом с Октябрьским»); подбор по месту — только по точке и радиусу.
  area text NOT NULL DEFAULT '' CHECK (char_length(area) <= 200),
  lat double precision CHECK (lat BETWEEN -90 AND 90),
  lng double precision CHECK (lng BETWEEN -180 AND 180),
  radius_km int CHECK (radius_km BETWEEN 1 AND 100),
  note text NOT NULL DEFAULT '' CHECK (char_length(note) <= 500),
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'closed')),
  -- Запрос живёт 30 дней: старый интерес не должен присылать уведомления вечно.
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((lat IS NULL) = (lng IS NULL)),
  CHECK (radius_km IS NULL OR lat IS NOT NULL)
);
-- Один активный запрос на направление: второй такой же лишь дублировал бы уведомления.
CREATE UNIQUE INDEX company_requests_one_active ON company_requests (user_id, sport_id) WHERE status = 'active';
CREATE INDEX company_requests_matching ON company_requests (sport_id, expires_at) WHERE status = 'active';

CREATE TABLE company_matches (
  request_id uuid NOT NULL REFERENCES company_requests(id) ON DELETE CASCADE,
  event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  -- notified — совпадение найдено при публикации события и человеку ушло уведомление;
  -- false — событие уже было на платформе, когда человек создал запрос (показываем списком).
  notified boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (request_id, event_id)
);
CREATE INDEX company_matches_event ON company_matches (event_id);

-- +goose Down
DROP TABLE company_matches;
DROP TABLE company_requests;
