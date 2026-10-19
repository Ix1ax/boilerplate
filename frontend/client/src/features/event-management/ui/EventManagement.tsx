import { useState } from 'react';

import { EventEditForm } from '@/features/event-edit';

import { toApiError } from '@/shared/api';
import { Badge, Button, Label, Notice, Textarea } from '@/shared/ui';

import { useEventManage } from '../model';
import { ParticipantRow } from './ParticipantRow';

interface EventManagementProps {
  eventId: string;
  onConfirmCancel: () => Promise<boolean>;
}

export const EventManagement = ({
  eventId: routeEventId,
  onConfirmCancel,
}: EventManagementProps) => {
  const [editing, setEditing] = useState(false);
  const {
    event,
    participants,
    feedback,
    reason,
    setReason,
    reasonError,
    action,
    cancel,
  } = useEventManage(routeEventId, onConfirmCancel);
  if (event.isPending)
    return (
      <div className="w-full" aria-busy="true">
        Загружаем встречу…
      </div>
    );
  if (event.isError || !event.data)
    return (
      <div className="w-full">
        <p role="alert">Не удалось загрузить встречу.</p>
        <Button onClick={() => void event.refetch()}>Повторить</Button>
      </div>
    );
  if (!event.data.isOrganizer)
    return (
      <div className="w-full">
        <h2 className="title-display text-[26px]">
          Управление доступно организатору
        </h2>
      </div>
    );
  const closed =
    event.data.status !== 'published' ||
    new Date(event.data.startsAt) <= new Date();
  return (
    <>
      <div className="w-full">
        <div className="max-w-[960px]">
          <h2 className="title-display text-[26px]">{event.data.title}</h2>
          <div className="mt-6 flex flex-wrap items-center gap-4">
            <Badge>В составе: {event.data.confirmedCount}</Badge>
            <Badge>В очереди: {event.data.waitlistCount}</Badge>
            {!closed && (
              <Button
                size="lg"
                variant="outline"
                disabled={action.isPending}
                onClick={() => action.mutate({ kind: 'remind' })}
              >
                Напомнить участникам
              </Button>
            )}
            {!closed && (
              <Button
                size="lg"
                variant="outline"
                onClick={() => setEditing((value) => !value)}
              >
                {editing ? 'Скрыть редактор' : 'Изменить встречу'}
              </Button>
            )}
          </div>
          <p className="mt-3 text-sm text-muted-foreground">
            Напоминание можно отправить до начала встречи, не чаще раза в час.
          </p>
          {!closed && editing && (
            <section className="mt-8">
              <h2 className="title-display text-[26px]">Редактор встречи</h2>
              <p className="mt-2 text-sm text-muted-foreground">
                Направление, режим записи и «от чьего имени» не меняются.
                Перенос времени или места — участники получат уведомление.
              </p>
              <div className="mt-5">
                <EventEditForm event={event.data} />
              </div>
            </section>
          )}
          {feedback && (
            <Notice role="status" tone="success" className="mt-5">
              {feedback}
            </Notice>
          )}
          {action.isError && (
            <Notice role="alert" tone="danger" className="mt-5">
              {toApiError(action.error).message}
            </Notice>
          )}
          <section className="mt-8">
            <h2 className="title-display text-[26px]">Участники и заявки</h2>
            {participants.isPending && (
              <p className="mt-4" aria-busy="true">
                Загружаем состав…
              </p>
            )}
            {participants.isError && (
              <div className="mt-4">
                <p role="alert">Не удалось загрузить участников.</p>
                <Button onClick={() => void participants.refetch()}>
                  Повторить
                </Button>
              </div>
            )}
            {!participants.isPending &&
              !participants.isError &&
              Boolean(participants.data?.length) && (
                <ul className="mt-5 grid gap-4">
                  {participants.data?.map((participant) => (
                    <ParticipantRow
                      key={participant.id}
                      participant={participant}
                      busy={action.isPending || closed}
                      onDecision={(participantId, accept, comment) =>
                        action.mutate({
                          kind: accept ? 'approve' : 'reject',
                          participantId,
                          reason: comment,
                        })
                      }
                    />
                  ))}
                </ul>
              )}
            {!participants.isPending &&
              !participants.isError &&
              !participants.data?.length && (
                <p className="mt-4 rounded-3xl bg-card p-6">
                  Пока никто не записался. Поделитесь ссылкой на встречу.
                </p>
              )}
          </section>
          {!closed && (
            <section className="mt-10 grid gap-4 rounded-3xl bg-card p-6">
              <h2 className="title-display text-[26px]">Отменить встречу</h2>
              <p>
                Все записавшиеся увидят причину и получат уведомление. Отмену
                нельзя вернуть.
              </p>
              <Label htmlFor="cancel-reason">Причина отмены</Label>
              <Textarea
                id="cancel-reason"
                value={reason}
                maxLength={1000}
                aria-invalid={Boolean(reasonError)}
                aria-describedby={reasonError ? 'cancel-error' : undefined}
                onChange={(e) => setReason(e.target.value)}
              />
              {reasonError && (
                <p id="cancel-error" role="alert" className="text-danger">
                  {reasonError}
                </p>
              )}
              <Button
                size="lg"
                variant="outline"
                disabled={action.isPending}
                onClick={() => void cancel()}
              >
                Отменить встречу и уведомить участников
              </Button>
            </section>
          )}
        </div>
      </div>
    </>
  );
};
