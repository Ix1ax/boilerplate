-- +goose Up
CREATE TABLE friendships (
  user_a text NOT NULL REFERENCES profiles(user_id),
  user_b text NOT NULL REFERENCES profiles(user_id),
  requested_by text NOT NULL,
  status text NOT NULL CHECK (status IN ('pending','accepted','declined')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(user_a,user_b),
  CHECK(user_a < user_b), CHECK(requested_by IN(user_a,user_b))
);
CREATE INDEX friendships_b_idx ON friendships(user_b,status);
CREATE TABLE direct_chats (
  id uuid PRIMARY KEY,
  user_a text NOT NULL REFERENCES profiles(user_id),
  user_b text NOT NULL REFERENCES profiles(user_id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(user_a,user_b), CHECK(user_a < user_b)
);
CREATE TABLE direct_messages (
  id uuid PRIMARY KEY,
  chat_id uuid NOT NULL REFERENCES direct_chats(id),
  author_id text NOT NULL REFERENCES profiles(user_id),
  body text NOT NULL CHECK(char_length(btrim(body)) BETWEEN 1 AND 2000),
  created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX direct_messages_page_idx ON direct_messages(chat_id,created_at DESC,id DESC);
CREATE TABLE event_reviews (
  id uuid PRIMARY KEY,
  event_id uuid NOT NULL REFERENCES events(id),
  author_id text NOT NULL REFERENCES profiles(user_id),
  organizer_id text NOT NULL,
  score int NOT NULL CHECK(score BETWEEN 1 AND 5),
  body text NOT NULL CHECK(char_length(btrim(body)) BETWEEN 1 AND 2000),
  hidden_at timestamptz,
  hidden_by text,
  hidden_reason text NOT NULL DEFAULT '' CHECK(char_length(hidden_reason)<=500),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(event_id,author_id),
  CHECK(hidden_at IS NULL OR btrim(hidden_reason)<>'')
);
CREATE INDEX event_reviews_profile_idx ON event_reviews(organizer_id,created_at DESC,id DESC) WHERE hidden_at IS NULL;
CREATE TABLE stories (
  id uuid PRIMARY KEY,
  author_id text NOT NULL REFERENCES profiles(user_id),
  body text NOT NULL CHECK(char_length(btrim(body)) BETWEEN 1 AND 1000),
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL DEFAULT now()+interval '24 hours',
  media_key text,
  media_content_type text,
  media_size bigint,
  deleted_at timestamptz,
  deleted_by text,
  delete_reason text NOT NULL DEFAULT '' CHECK(char_length(delete_reason)<=500)
);
CREATE INDEX stories_feed_idx ON stories(created_at DESC,id DESC) WHERE deleted_at IS NULL;
CREATE TABLE organization_members (
  organization_id uuid NOT NULL REFERENCES organizations(id),
  user_id text NOT NULL REFERENCES profiles(user_id),
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(organization_id,user_id)
);
CREATE TABLE story_media_objects (
  object_key text PRIMARY KEY,
  story_id uuid NOT NULL REFERENCES stories(id),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX story_media_cleanup_idx ON story_media_objects(created_at);
-- +goose Down
DROP TABLE story_media_objects,organization_members,stories,event_reviews,direct_messages,direct_chats,friendships;
