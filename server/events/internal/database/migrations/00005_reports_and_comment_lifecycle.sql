-- +goose Up
-- Автор может удалить своё сообщение: текст пропадает у всех, кроме админа, ответы остаются.
ALTER TABLE comments DROP CONSTRAINT comments_status_check;
ALTER TABLE comments ADD CONSTRAINT comments_status_check CHECK (status IN ('visible', 'hidden', 'deleted'));

-- Жалобы пользователей на мероприятие или сообщение. Решает администратор: скрывает/снимает
-- (жалобы на цель закрываются автоматически в той же транзакции) или отклоняет с пояснением.
CREATE TABLE reports (
  id uuid PRIMARY KEY,
  target_type text NOT NULL CHECK (target_type IN ('event', 'comment')),
  target_id uuid NOT NULL,
  -- Событие, к которому относится цель: из жалобы на сообщение админ сразу видит обсуждение.
  event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  reporter_id text NOT NULL,
  reason text NOT NULL CHECK (reason IN ('spam', 'offensive', 'fraud', 'wrong_info', 'dangerous', 'other')),
  details text NOT NULL DEFAULT '' CHECK (char_length(details) <= 1000),
  status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved', 'dismissed')),
  resolution text NOT NULL DEFAULT '' CHECK (char_length(resolution) <= 1000),
  resolved_by text,
  resolved_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (reason <> 'other' OR btrim(details) <> ''),
  CHECK (status = 'open' OR resolved_at IS NOT NULL)
);
-- Одна открытая жалоба человека на цель: повторное нажатие не создаёт дубль.
CREATE UNIQUE INDEX reports_open_unique ON reports (target_type, target_id, reporter_id) WHERE status = 'open';
CREATE INDEX reports_queue_idx ON reports (status, created_at);
CREATE INDEX reports_target_idx ON reports (target_type, target_id, status);
CREATE INDEX reports_reporter_idx ON reports (reporter_id, created_at DESC);

-- +goose Down
DROP TABLE reports;
UPDATE comments SET status = 'hidden', hidden_reason = 'Удалено автором' WHERE status = 'deleted';
ALTER TABLE comments DROP CONSTRAINT comments_status_check;
ALTER TABLE comments ADD CONSTRAINT comments_status_check CHECK (status IN ('visible', 'hidden'));
