import { useState } from 'react';
import { Link } from 'react-router';
import { useQuery } from '@tanstack/react-query';

import {
  COMMENT_KEYS,
  Discussion,
  getComments,
} from '@/features/event-discussion';
import { EventEditForm } from '@/features/event-edit';

import { toApiError } from '@/shared/api';
import { DYNAMIC_ROUTES } from '@/shared/constants';
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
  const [editing, setEditing] = useState(
    new URLSearchParams(window.location.search).get('edit') === '1',
  );
  const [tab, setTab] = useState('pending');
  const [draft, setDraft] = useState('');
  const comments = useQuery({
    queryKey: COMMENT_KEYS.LIST(routeEventId),
    queryFn: () => getComments(routeEventId),
    meta: { silent: true },
  });
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
  const unanswered = (comments.data ?? []).filter(
    (comment) =>
      comment.status === 'visible' &&
      !comment.parentId &&
      !comment.isOrganizer &&
      !(comments.data ?? []).some(
        (reply) =>
          reply.status === 'visible' &&
          reply.parentId === comment.id &&
          reply.isOrganizer,
      ),
  ).length;
  const visibleParticipants = (participants.data ?? []).filter(
    (participant) => participant.status === tab,
  );
  return (
    <>
      <div className="w-full">
        <div className="max-w-[960px]">
          <h2 className="title-display text-[26px]">{event.data.title}</h2>
          <div className="mt-5 grid grid-cols-2 gap-3 sm:grid-cols-4">
            {[
              ['В составе', event.data.confirmedCount],
              ['Ждут решения', event.data.pendingCount],
              ['В листе ожидания', event.data.waitlistCount],
              ['Вопросы без ответа', unanswered],
            ].map(([label, value]) => (
              <div key={String(label)} className="rounded-3xl bg-card p-5">
                <b className="text-3xl">{value}</b>
                <p>{label}</p>
              </div>
            ))}
          </div>
          <div className="mt-6 flex flex-wrap items-center gap-4">
            <Button
              variant="outline"
              render={
                <Link
                  to={`${DYNAMIC_ROUTES.EVENT(event.data.id)}#discussion`}
                />
              }
            >
              Открыть обсуждение
            </Button>
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
          <nav
            className="mt-8 flex flex-wrap gap-3"
            aria-label="Разделы управления"
          >
            {[
              ['pending', 'Заявки'],
              ['confirmed', 'Состав'],
              ['waitlist', 'Лист ожидания'],
              ['rejected', 'Отклонённые'],
              ['discussion', 'Обсуждение'],
            ].map(([value, label]) => (
              <Button
                key={value}
                variant={tab === value ? 'default' : 'outline'}
                onClick={() => setTab(value)}
                aria-pressed={tab === value}
              >
                {label}
              </Button>
            ))}
          </nav>
          {tab === 'discussion' ? (
            <section className="mt-5">
              <Discussion
                eventId={routeEventId}
                isOrganizer
                closed={closed}
                draft={draft}
                onDraftChange={setDraft}
              />
            </section>
          ) : (
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
                visibleParticipants.length > 0 && (
                  <ul className="mt-5 grid gap-4">
                    {visibleParticipants.map((participant) => (
                      <ParticipantRow
                        key={participant.id}
                        participant={participant}
                        busy={action.isPending || closed}
                        onRemove={(participantId, reason) =>
                          action.mutate({
                            kind: 'remove',
                            participantId,
                            reason,
                          })
                        }
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
                visibleParticipants.length === 0 && (
                  <p className="mt-4 rounded-3xl bg-card p-6">
                    В этой вкладке пока нет участников или заявок.
                  </p>
                )}
            </section>
          )}
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
