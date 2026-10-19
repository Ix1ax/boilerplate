-- +goose Up
-- Администраторы панели отделены от пользователей продукта: свой вход, своя сессия в gateway.
-- Роль в токене «Столото ID» и обычная регистрация доступа к админке не дают.
CREATE TABLE admins (
  id uuid PRIMARY KEY,
  email text NOT NULL CHECK (char_length(email) BETWEEN 3 AND 254 AND email = lower(email)),
  display_name text NOT NULL CHECK (char_length(btrim(display_name)) BETWEEN 1 AND 100),
  -- Хеш Argon2id в формате PHC, как у учётных записей пользователей.
  password_hash text NOT NULL,
  role text NOT NULL DEFAULT 'admin' CHECK (role IN ('superadmin', 'admin')),
  -- deleted — мягкое удаление: история решений админа остаётся, почта освобождается.
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'blocked', 'deleted')),
  block_reason text NOT NULL DEFAULT '' CHECK (char_length(block_reason) <= 500),
  failed_logins int NOT NULL DEFAULT 0,
  locked_until timestamptz,
  last_login_at timestamptz,
  password_changed_at timestamptz NOT NULL DEFAULT now(),
  created_by uuid REFERENCES admins(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CHECK (status <> 'blocked' OR btrim(block_reason) <> ''),
  -- Суперадминистратор всегда активен: иначе управлять админами станет некому.
  CHECK (role = 'admin' OR status = 'active')
);
CREATE UNIQUE INDEX admins_email_key ON admins (email) WHERE status <> 'deleted';
-- Суперадминистратор один: его создаёт сервис при старте из SUPERADMIN_EMAIL и SUPERADMIN_PASSWORD.
CREATE UNIQUE INDEX admins_one_superadmin ON admins (role) WHERE role = 'superadmin';
CREATE INDEX admins_created_idx ON admins (created_at DESC);

-- История: кто создал, заблокировал, удалил админа и сбросил ему пароль.
CREATE TABLE admin_history (
  id bigserial PRIMARY KEY,
  admin_id uuid NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
  action text NOT NULL CHECK (action IN ('created', 'blocked', 'unblocked', 'password_changed', 'password_reset', 'deleted')),
  actor text NOT NULL,
  comment text NOT NULL DEFAULT '' CHECK (char_length(comment) <= 500),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admin_history_admin_idx ON admin_history (admin_id, created_at);

-- +goose Down
DROP TABLE admin_history;
DROP TABLE admins;
