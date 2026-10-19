-- +goose Up
-- Дополнения к основному пути: «Соберём состав», «Встречу новичка», напоминания с погодой.

-- «Соберём состав»: игра состоится, только если к дедлайну наберётся минимум подтверждённых
-- участников (организатор не считается). quorum_status: none — без минимума; gathering — набираем;
-- reached — состав собран (больше не откатывается, даже если кто-то отменит запись: участникам
-- уже объявили, что игра будет); failed — не набрали, событие отменено автоматически.
ALTER TABLE events
  ADD COLUMN min_participants int CHECK (min_participants IS NULL OR min_participants BETWEEN 2 AND 1000),
  ADD COLUMN quorum_deadline timestamptz,
  ADD COLUMN quorum_status text NOT NULL DEFAULT 'none' CHECK (quorum_status IN ('none', 'gathering', 'reached', 'failed')),
  -- outdoor — занятие на улице: для него показываем погоду и советы, что надеть.
  ADD COLUMN outdoor boolean NOT NULL DEFAULT false,
  ADD CONSTRAINT events_quorum_check CHECK (
    (min_participants IS NULL AND quorum_deadline IS NULL AND quorum_status = 'none')
    OR (min_participants IS NOT NULL AND quorum_deadline IS NOT NULL AND quorum_deadline <= starts_at)
  ),
  ADD CONSTRAINT events_quorum_capacity CHECK (capacity IS NULL OR min_participants IS NULL OR min_participants <= capacity);
CREATE INDEX events_quorum_due ON events (quorum_deadline) WHERE quorum_status = 'gathering';

-- «Встречу новичка»: first_time — человек идёт впервые; buddy — опытный участник готов его встретить.
-- buddy_id — кто встречает этого новичка; проводник отвечает максимум за трёх (см. gatherings.go).
ALTER TABLE participations
  ADD COLUMN first_time boolean NOT NULL DEFAULT false,
  ADD COLUMN buddy boolean NOT NULL DEFAULT false,
  ADD COLUMN buddy_id uuid REFERENCES participations(id) ON DELETE SET NULL,
  ADD CONSTRAINT participations_buddy_not_newcomer CHECK (NOT (first_time AND buddy)),
  ADD CONSTRAINT participations_buddy_not_self CHECK (buddy_id IS NULL OR buddy_id <> id);
CREATE INDEX participations_buddy ON participations (buddy_id) WHERE buddy_id IS NOT NULL;

-- Отправленные напоминания: строка появляется в той же транзакции, что и уведомления,
-- поэтому повторный запуск задания или вторая реплика не пришлют напоминание дважды.
CREATE TABLE event_reminders (
  event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  -- day — за сутки, soon — за два часа, manual — организатор нажал «Напомнить» (не чаще раза в час).
  kind text NOT NULL CHECK (kind IN ('day', 'soon', 'manual')),
  sent_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (event_id, kind)
);

-- +goose Down
DROP TABLE event_reminders;
DROP INDEX participations_buddy;
ALTER TABLE participations
  DROP CONSTRAINT participations_buddy_not_self,
  DROP CONSTRAINT participations_buddy_not_newcomer,
  DROP COLUMN buddy_id,
  DROP COLUMN buddy,
  DROP COLUMN first_time;
DROP INDEX events_quorum_due;
ALTER TABLE events
  DROP CONSTRAINT events_quorum_capacity,
  DROP CONSTRAINT events_quorum_check,
  DROP COLUMN outdoor,
  DROP COLUMN quorum_status,
  DROP COLUMN quorum_deadline,
  DROP COLUMN min_participants;
