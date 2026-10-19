import { useState } from 'react';
import { Link } from 'react-router';

import { useConfirm } from '@/features/confirm';

import {
  type Event,
  formatDayShort,
  formatTeamCount,
  formatTime,
  isPast,
} from '@/entities/event';
import { useCurrentUser } from '@/entities/user';

import { ApiError } from '@/shared/api';
import { DYNAMIC_ROUTES, STATIC_ROUTES } from '@/shared/constants';
import { PopIn, Spin, StateSwap } from '@/shared/motion';
import {
  Badge,
  Button,
  Checkbox,
  Label,
  Notice,
  RadarRings,
  Textarea,
} from '@/shared/ui';

import { useJoinEvent, useLeaveEvent } from '../model';
import { useBuddy } from '../model/useJoin';

interface JoinPanelProps {
  event: Event;
  onRefresh: () => void;
}

/**
 * Все состояния записи в одном месте: записаться, заявка, лист ожидания,
 * вы в составе, решение организатора, снято или прошло, вы организатор.
 */
export const JoinPanel = ({ event, onRefresh }: JoinPanelProps) => {
  const {data:user,isPending:checkingUser} = useCurrentUser();
  const join = useJoinEvent();
  const buddy = useBuddy();
  const leave = useLeaveEvent();
  const confirm = useConfirm();
  const [firstTime, setFirstTime] = useState(false);
  const [message, setMessage] = useState('');

  const me = event.myParticipation;
  const status = me?.status;
  const isFree = event.joinMode === 'open';
  const full = event.spotsLeft === 0;
  const conflict = join.error instanceof ApiError && join.error.status === 409;

  const submit = () =>
    join.mutate({
      eventId: event.id,
      firstTime,
      message: isFree ? undefined : message.trim(),
    });

  const cancel = async () => {
    const ok = await confirm({
      title:
        status === 'pending'
          ? 'Отозвать заявку?'
          : 'Отменить запись на встречу?',
      description:
        status === 'confirmed' && full
          ? 'Ваше место сразу получит первый из листа ожидания.'
          : 'Записаться снова можно, пока встреча опубликована.',
      confirmText: status === 'pending' ? 'Отозвать' : 'Отменить запись',
      cancelText: 'Оставить',
      destructive: true,
    });
    if (ok) leave.mutate(event.id);
  };

  let hint = 'Организатор рассмотрит заявку. До решения вы не в составе.';
  let submitLabel = 'Отправить заявку';
  if (isFree) {
    hint = full
      ? 'Мест нет: встанете в лист ожидания и получите место, если кто-то откажется.'
      : 'Свободная запись: после нажатия вы сразу в составе.';
    submitLabel = full ? 'Встать в лист ожидания' : 'Записаться';
  }

  let stateKey = 'idle';
  if (event.status !== 'published') stateKey = 'closed';
  else if (isPast(event)) stateKey = 'past';
  else if (event.isOrganizer) stateKey = 'organizer';
  else if (conflict) stateKey = 'conflict';
  else if (join.isPending) stateKey = 'sending';
  else if (status) stateKey = status;

  const renderState = () => {
    if(checkingUser) return <p role="status">Проверяем сессию…</p>;
    if(!user) return <div className="grid gap-3"><h2 className="title-display text-2xl">Присоединяйтесь к встрече</h2><p>Войдите, чтобы записаться, отправить заявку или задать вопрос.</p><Button render={<Link to={`${STATIC_ROUTES.LOGIN}?next=${encodeURIComponent(DYNAMIC_ROUTES.EVENT(event.id))}`} />}>Войти и записаться</Button></div>;

    if (event.status !== 'published') {
      return (
        <Notice
          tone="neutral"
          title={
            event.status === 'cancelled'
              ? '⊘ Встреча отменена'
              : '⊘ Снята с публикации'
          }
        >
          <span>Записаться нельзя. Причина — вверху страницы.</span>
        </Notice>
      );
    }
    if (isPast(event)) {
      return (
        <Notice tone="neutral" title="Встреча уже прошла">
          <Link
            to={STATIC_ROUTES.EVENTS}
            className="flex min-h-11 items-center font-bold"
          >
            Посмотреть ближайшие встречи
          </Link>
        </Notice>
      );
    }
    if (event.isOrganizer) {
      return (
        <div className="flex flex-col gap-2.5">
          <Badge variant="gold" className="text-[15px]">
            Вы организатор
          </Badge>
          <span>
            {formatTeamCount(event)}
            {event.pendingCount > 0 && ` · заявок: ${event.pendingCount}`}.
            Заявки, состав и ответы — в управлении встречей.
          </span>
          <Button
            size="lg"
            render={<Link to={DYNAMIC_ROUTES.EVENT_MANAGE(event.id)} />}
          >
            Управлять встречей
          </Button>
        </div>
      );
    }
    if (conflict) {
      return (
        <Notice role="alert" tone="outline" className="gap-2.5">
          <b className="text-lg">Состояние уже изменилось. Обновите страницу</b>
          <span>Пока страница была открыта, данные встречи поменялись.</span>
          <Button
            variant="outline"
            onClick={() => {
              join.reset();
              onRefresh();
            }}
          >
            Обновить
          </Button>
        </Notice>
      );
    }
    if (join.isPending) {
      return (
        <Button size="lg" disabled>
          <Spin />
          {isFree ? 'Записываем…' : 'Отправляем…'}
        </Button>
      );
    }
    if (status === 'confirmed') {
      return (
        <Notice
          tone="success"
          title={
            <>
              <PopIn>✓</PopIn> Вы записаны
            </>
          }
          className="gap-2.5"
        >
          <span>Организатор видит вас в составе. Встреча — в «Мой спорт».</span>
          {me?.buddy && (
            <span>
              Вас встретит <b>{me.buddy.name}</b> — подойдите к нему на месте.
            </span>
          )}
          {!me?.firstTime && (
            <Button
              variant="outline"
              disabled={buddy.isPending}
              onClick={() =>
                buddy.mutate({ id: event.id, enabled: !me?.isBuddy })
              }
            >
              {me?.isBuddy
                ? 'Больше не встречаю новичков'
                : 'Готов встретить новичка'}
            </Button>
          )}
          {me?.newcomers?.length ? (
            <p>
              Вы встречаете:{' '}
              {me.newcomers.map((person) => person.name).join(', ')}.
            </p>
          ) : null}
          <span className="text-[15px]">
            Напомним за сутки и за 2 часа до начала.
          </span>
          <Button
            variant="outline"
            size="sm"
            className="self-start"
            disabled={leave.isPending}
            onClick={() => void cancel()}
          >
            Отменить запись
          </Button>
        </Notice>
      );
    }
    if (status === 'pending') {
      return (
        <Notice
          tone="warning"
          title="◷ Ожидает решения организатора"
          className="gap-2.5"
        >
          <span>
            Вы пока не в составе. Решение придёт в уведомления и появится в «Мой
            спорт».
          </span>
          <Button
            variant="outline"
            size="sm"
            className="self-start"
            disabled={leave.isPending}
            onClick={() => void cancel()}
          >
            Отозвать заявку
          </Button>
        </Notice>
      );
    }
    if (status === 'waitlist') {
      return (
        <Notice
          tone="warning"
          title="◷ Вы в листе ожидания"
          className="gap-2.5"
        >
          <span>
            Мест пока нет. Как только кто-то откажется, место получит первый в
            очереди — мы сообщим.
          </span>
          <Button
            variant="outline"
            size="sm"
            className="self-start"
            disabled={leave.isPending}
            onClick={() => void cancel()}
          >
            Выйти из очереди
          </Button>
        </Notice>
      );
    }
    if (status === 'rejected') {
      return (
        <Notice
          tone="danger"
          title="✕ Организатор отклонил заявку"
          className="gap-1.5"
        >
          <span>
            Вы не в составе.
            {me?.decisionReason && <> Причина: «{me.decisionReason}».</>}
          </span>
          <Link
            to={STATIC_ROUTES.EVENTS}
            className="flex min-h-11 items-center font-bold"
          >
            Посмотреть другие встречи
          </Link>
        </Notice>
      );
    }

    return (
      <div className="flex flex-col gap-3">
        <span>{hint}</span>
        {!isFree && (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="join-message">Пара слов организатору</Label>
            <Textarea
              id="join-message"
              rows={2}
              maxLength={500}
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Например: играю два года, есть своя форма"
            />
          </div>
        )}
        <label className="flex min-h-11 items-center gap-2.5 font-semibold">
          <Checkbox checked={firstTime} onCheckedChange={setFirstTime} />
          Иду впервые — пусть меня встретят
        </label>
        {join.isError && !conflict && (
          <span role="alert" className="font-semibold text-danger">
            ! Не получилось: {join.error.message}. Попробуйте ещё раз.
          </span>
        )}
        <Button size="lg" onClick={submit}>
          {submitLabel}
        </Button>
      </div>
    );
  };

  return (
    <div className="relative flex flex-col gap-4 overflow-hidden rounded-3xl bg-card p-6 shadow-[0_30px_60px_-34px_rgb(18_18_18/45%)]">
      <RadarRings
        tone="blue"
        sizes={[170]}
        strength={0.22}
        className="top-6 right-6"
      />
      <span
        aria-hidden="true"
        className="absolute -top-5 -right-5 size-[90px] rounded-full bg-gold"
      />
      <div className="relative flex flex-col gap-0.5 pr-16">
        <span className="font-mono text-[13px] text-muted-foreground uppercase">
          {formatDayShort(event.startsAt)} · {formatTime(event.startsAt)}
        </span>
        <b className="text-[22px] leading-snug">{event.location}</b>
      </div>
      <div aria-live="polite" className="relative">
        <StateSwap stateKey={stateKey} className="flex flex-col gap-3">
          {renderState()}
        </StateSwap>
      </div>
    </div>
  );
};
