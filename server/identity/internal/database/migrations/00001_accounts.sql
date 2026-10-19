-- +goose Up
-- Учётные записи обычной регистрации (почта и пароль). Пользователи «Столото ID» живут в Keycloak;
-- id здесь — такой же UUID, как sub в токене, поэтому сервисы не различают способы входа.
CREATE TABLE accounts (
  id uuid PRIMARY KEY,
  -- Почта хранится в нижнем регистре: уникальность и вход не зависят от написания.
  email text NOT NULL CHECK (char_length(email) BETWEEN 3 AND 254 AND email = lower(email)),
  display_name text NOT NULL CHECK (char_length(btrim(display_name)) BETWEEN 1 AND 100),
  -- Хеш Argon2id в формате PHC: параметры хранятся вместе с солью и переживают их смену.
  password_hash text NOT NULL,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'blocked')),
  -- Причину блокировки пользователь видит при попытке входа; без неё заблокировать нельзя.
  block_reason text NOT NULL DEFAULT '' CHECK (char_length(block_reason) <= 500),
  blocked_by text,
  blocked_at timestamptz,
  -- Неудачные попытки подряд: после порога вход временно закрыт (подбор пароля).
  failed_logins int NOT NULL DEFAULT 0,
  locked_until timestamptz,
  last_login_at timestamptz,
  password_changed_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (status <> 'blocked' OR btrim(block_reason) <> '')
);
CREATE UNIQUE INDEX accounts_email_key ON accounts (email);
CREATE INDEX accounts_created_idx ON accounts (created_at DESC);

-- История блокировок: кто, когда и почему — для разбора спорных случаев.
CREATE TABLE account_history (
  id bigserial PRIMARY KEY,
  account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  action text NOT NULL CHECK (action IN ('registered', 'blocked', 'unblocked', 'password_changed')),
  actor text NOT NULL,
  comment text NOT NULL DEFAULT '' CHECK (char_length(comment) <= 500),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX account_history_account_idx ON account_history (account_id, created_at);

-- +goose Down
DROP TABLE account_history;
DROP TABLE accounts;
