-- +goose Up
-- Привязка аккаунта к пользователю мессенджера MAX: один MAX-аккаунт — один пользователь.
CREATE TABLE max_links (
  user_id text PRIMARY KEY,
  max_user_id bigint NOT NULL UNIQUE,
  linked_at timestamptz NOT NULL DEFAULT now()
);
-- Одноразовые коды для deep link https://max.ru/<бот>/start/<код>, живут 10 минут.
CREATE TABLE max_link_codes (
  code text PRIMARY KEY,
  user_id text NOT NULL,
  expires_at timestamptz NOT NULL
);
-- +goose Down
DROP TABLE max_link_codes;
DROP TABLE max_links;
