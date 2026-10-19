-- +goose Up
CREATE TABLE recommendation_preferences (
  user_id text PRIMARY KEY REFERENCES profiles(user_id),
  sport_ids uuid[] NOT NULL DEFAULT '{}',
  level text NOT NULL DEFAULT 'any' CHECK (level IN ('any','beginner','amateur','pro')),
  city text NOT NULL DEFAULT '' CHECK (char_length(city) <= 100),
  weekdays int[] NOT NULL DEFAULT '{}',
  day_parts text[] NOT NULL DEFAULT '{}',
  lat double precision CHECK (lat BETWEEN -90 AND 90),
  lng double precision CHECK (lng BETWEEN -180 AND 180),
  radius_km int NOT NULL DEFAULT 20 CHECK (radius_km BETWEEN 1 AND 100),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((lat IS NULL) = (lng IS NULL)),
  CHECK (weekdays <@ ARRAY[1,2,3,4,5,6,7]),
  CHECK (day_parts <@ ARRAY['morning','day','evening']::text[]),
  CHECK (cardinality(sport_ids) <= 30)
);
CREATE TABLE recommendation_feedback (
  user_id text NOT NULL REFERENCES profiles(user_id),
  event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  action text NOT NULL CHECK (action IN ('view','interested','dismiss')),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id,event_id)
);
CREATE INDEX recommendation_feedback_event_idx ON recommendation_feedback(event_id,action);
-- +goose Down
DROP TABLE recommendation_feedback;
DROP TABLE recommendation_preferences;
