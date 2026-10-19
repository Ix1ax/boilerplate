-- +goose Up
-- Жалобы на пользовательский контент вне обсуждений: отзывы, истории с фото и видео, личные сообщения.
ALTER TABLE reports DROP CONSTRAINT reports_target_type_check;
ALTER TABLE reports ADD CONSTRAINT reports_target_type_check
  CHECK (target_type IN ('event', 'comment', 'review', 'story', 'direct_message'));
-- У истории и личного сообщения нет мероприятия; у отзыва оно есть и нужно админу для контекста.
ALTER TABLE reports ALTER COLUMN event_id DROP NOT NULL;
ALTER TABLE reports ADD CONSTRAINT reports_event_required
  CHECK (target_type NOT IN ('event', 'comment', 'review') OR event_id IS NOT NULL);

-- Админ скрывает только то личное сообщение, на которое пожаловались: ленты чатов в админке нет.
ALTER TABLE direct_messages ADD COLUMN hidden_at timestamptz;
ALTER TABLE direct_messages ADD COLUMN hidden_by text;
ALTER TABLE direct_messages ADD COLUMN hidden_reason text NOT NULL DEFAULT ''
  CHECK (char_length(hidden_reason) <= 500);

-- +goose Down
ALTER TABLE direct_messages DROP COLUMN hidden_reason, DROP COLUMN hidden_by, DROP COLUMN hidden_at;
DELETE FROM reports WHERE target_type IN ('review', 'story', 'direct_message');
ALTER TABLE reports DROP CONSTRAINT reports_event_required;
ALTER TABLE reports ALTER COLUMN event_id SET NOT NULL;
ALTER TABLE reports DROP CONSTRAINT reports_target_type_check;
ALTER TABLE reports ADD CONSTRAINT reports_target_type_check CHECK (target_type IN ('event', 'comment'));
