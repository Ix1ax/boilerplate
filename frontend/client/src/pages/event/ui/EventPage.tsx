import { useState } from 'react';
import { Link, useParams } from 'react-router';
import { ArrowLeftIcon } from 'lucide-react';

import { Discussion, DISCUSSION_INPUT_ID } from '@/features/event-discussion';
import { JoinPanel } from '@/features/event-join';
import { ReportButton } from '@/features/report';

import {
  type Event,
  formatDayShort,
  formatTime,
  isPast,
  JOIN_MODE_LABELS,
  useEvent,
} from '@/entities/event';
import { getSportVisual, SportGlyph } from '@/entities/sport';
import { useCurrentUser } from '@/entities/user';

import { STATIC_ROUTES } from '@/shared/constants';
import { Badge, Button, Notice } from '@/shared/ui';

import { EventFacts } from './EventFacts';
import { EventWeatherNote } from './EventWeatherNote';
import { FirstMeeting } from './FirstMeeting';

const EventHero = ({ event }: { event: Event }) => {
  const { icon, cover } = getSportVisual(event.sport.name);
  return (
    <section
      className="relative overflow-hidden"
      style={{ background: cover.bg }}
    >
      <span
        aria-hidden="true"
        className="pointer-events-none absolute top-1/2 right-[calc(12%+80px)] size-0 max-md:right-10"
      >
        {[160, 300, 460, 640].map((size) => (
          <span
            key={size}
            className="absolute -translate-1/2 rounded-full border-[1.5px]"
            style={{ width: size, height: size, borderColor: cover.ring }}
          />
        ))}
        <SportGlyph
          icon={icon}
          color={cover.ink}
          size={96}
          className="absolute -translate-1/2 max-md:hidden"
        />
      </span>
      <div
        className="relative mx-auto max-w-[1280px] px-4 pt-7 pb-11 sm:px-8"
        style={{ color: cover.text }}
      >
        <Link
          to={STATIC_ROUTES.EVENTS}
          className="inline-flex min-h-11 items-center gap-2 font-bold"
        >
          <ArrowLeftIcon aria-hidden className="size-5" />
          Все встречи
        </Link>
        <div className="mt-[18px] flex flex-wrap gap-2">
          <Badge variant="ink" className="px-3.5 py-1.5 text-[15px] font-bold">
            {event.sport.name}
          </Badge>
          <Badge
            className={`border-[1.5px] border-ink bg-white px-3.5 py-1.5 text-[15px] font-bold text-ink ${event.joinMode === 'request' ? 'border-dashed' : ''}`}
          >
            {JOIN_MODE_LABELS[event.joinMode]}
          </Badge>
          {event.sport.isCustom && (
            <Badge variant="gold" className="px-3.5 py-1.5 text-[15px]">
              Новое направление
            </Badge>
          )}
          {event.outdoor && (
            <Badge className="bg-white px-3.5 py-1.5 text-[15px] font-bold text-ink">
              На улице
            </Badge>
          )}
        </div>
        <h1 className="display-caps mt-4 max-w-[820px] text-[clamp(36px,4.6vw,64px)] leading-[.95] tracking-[-0.035em]">
          {event.title}
        </h1>
      </div>
    </section>
  );
};

const QuorumNote = ({ event }: { event: Event }) => {
  if (!event.minParticipants || event.quorumStatus === 'none') return null;
  const left = Math.max(event.minParticipants - event.confirmedCount, 0);
  if (event.quorumStatus === 'reached')
    return <Badge variant="success">✓ Состав собран — встреча состоится</Badge>;
  if (event.quorumStatus === 'failed')
    return (
      <Badge variant="destructive">
        ✕ Не собрали состав — встреча отменена
      </Badge>
    );
  return (
    <p className="max-w-[720px] text-lg">
      <b>Соберём состав:</b> встреча состоится, если запишутся минимум{' '}
      {event.minParticipants}. Осталось {left}
      {event.quorumDeadline &&
        ` — до ${formatDayShort(event.quorumDeadline).toLowerCase()}, ${formatTime(event.quorumDeadline)}`}
      .
    </p>
  );
};

const EventDetails = ({
  event,
  onRefresh,
}: {
  event: Event;
  onRefresh: () => void;
}) => {
  const [draft, setDraft] = useState('');
  const {data:user} = useCurrentUser();
  const closed = event.status !== 'published' || isPast(event);

  const ask = (topic: string) => {
    setDraft(`Подскажите, пожалуйста: ${topic.toLowerCase()}?`);
    // Поле появится в DOM в этом же кадре; фокус переводим после отрисовки.
    requestAnimationFrame(() => {
      const input = document.getElementById(DISCUSSION_INPUT_ID);
      input?.focus();
      input?.scrollIntoView({ block: 'center', behavior: 'smooth' });
    });
  };

  return (
    <div className="mx-auto w-full max-w-[1280px] px-4 pt-7 pb-18 sm:px-8">
      <div className="flex flex-wrap items-start gap-7">
        <div className="flex min-w-0 flex-[999_1_560px] flex-col gap-7">
          {event.status !== 'published' && (
            <Notice
              role="status"
              tone="ink"
              title={
                event.status === 'cancelled'
                  ? '⊘ Встреча отменена'
                  : '⊘ Встреча снята с публикации'
              }
              className="rounded-[22px] px-6 py-5"
            >
              <span className="text-paper/85">
                {event.moderationReason
                  ? `Причина: ${event.moderationReason}.`
                  : 'Причина не указана.'}{' '}
                Ваша запись сохранена. Новые записи закрыты.
              </span>
            </Notice>
          )}

          {event.status !== 'published' && (
            <div className="flex flex-wrap gap-3">
              <Button render={<Link to={STATIC_ROUTES.EVENTS} />}>
                Найти другую встречу
              </Button>
              <Button variant="outline" render={<Link to={STATIC_ROUTES.MY} />}>
                Мой спорт
              </Button>
            </div>
          )}
          <EventFacts event={event} />
          <div className="flex flex-wrap gap-2">
            {event.zone && <Badge>{event.zone}</Badge>}
            {event.stepFree && <Badge>Без ступеней / есть пандус</Badge>}
            {event.companionAllowed && <Badge>Можно с сопровождающим</Badge>}
          </div>
          {event.meetingHost && (
            <p>
              <b>Кто встретит:</b> {event.meetingHost}
            </p>
          )}
          {!!user && !event.isOrganizer && event.status === 'published' && (
            <ReportButton eventId={event.id} />
          )}

          {event.outdoor && !closed && <EventWeatherNote eventId={event.id} />}

          {event.description.trim() && (
            <section className="flex flex-col gap-2.5">
              <h2 className="title-display text-[26px]">О встрече</h2>
              <p className="max-w-[720px] text-lg leading-relaxed whitespace-pre-line">
                {event.description}
              </p>
            </section>
          )}

          <FirstMeeting
            event={event}
            canAsk={!!user && !event.isOrganizer && !closed}
            onAsk={ask}
          />

          <section className="flex flex-col gap-2.5">
            <h2 className="title-display text-[26px]">Как попасть в состав</h2>
            {event.joinMode === 'open' ? (
              <p className="max-w-[720px] text-lg leading-relaxed">
                <b>Свободная запись — сразу в составе.</b> Организатор увидит
                вас в списке. Напомним о встрече за сутки и за 2 часа.
                {event.capacity &&
                  ` Мест: ${event.capacity}; когда они закончатся, откроется лист ожидания.`}
              </p>
            ) : (
              <p className="max-w-[720px] text-lg leading-relaxed">
                <b>По заявке — организатор решает, кого принять.</b> До решения
                вы не в составе. Решение и причину отказа увидите в «Мой спорт».
              </p>
            )}
            <QuorumNote event={event} />
          </section>

          <section
            aria-labelledby="event-discussion"
            className="flex flex-col gap-3"
          >
            <div>
              <h2 id="event-discussion" className="title-display text-[26px]">
                Обсуждение
              </h2>
              <p className="mt-1 text-ink-soft dark:text-muted-foreground">
                Вопрос можно задать до записи. Ответ организатора выделен.
              </p>
            </div>
            <Discussion
              eventId={event.id}
              isOrganizer={event.isOrganizer}
              closed={closed}
              draft={draft}
              onDraftChange={setDraft}
            />
          </section>
        </div>

        <aside
          aria-label="Запись на встречу"
          className="flex min-w-0 flex-[1_1_340px] flex-col gap-3 lg:sticky lg:top-6"
        >
          <JoinPanel event={event} onRefresh={onRefresh} />
          {event.capacity != null && event.status === 'published' && (
            <p className="px-2 text-sm text-muted-foreground">
              {event.spotsLeft
                ? `Свободно мест: ${event.spotsLeft} из ${event.capacity}.`
                : `Все ${event.capacity} мест заняты${event.waitlistCount ? `, в очереди: ${event.waitlistCount}` : ''}.`}
            </p>
          )}
        </aside>
      </div>
    </div>
  );
};

export const EventPage = () => {
  const { eventId = '' } = useParams();
  const { data: event, isPending, isError, error, refetch } = useEvent(eventId);

  if (isPending) {
    return (
      <div
        className="mx-auto flex w-full max-w-[1280px] flex-wrap gap-6 px-4 py-7 sm:px-8"
        aria-busy="true"
      >
        <div className="h-[420px] flex-[999_1_560px] animate-pulse rounded-3xl bg-stone motion-reduce:animate-none dark:bg-muted" />
        <div className="h-[300px] flex-[1_1_340px] animate-pulse rounded-3xl bg-stone motion-reduce:animate-none dark:bg-muted" />
      </div>
    );
  }

  if (isError || !event) {
    const notFound = error?.status === 404;
    return (
      <div className="mx-auto w-full max-w-[1280px] px-4 py-10 sm:px-8">
        <div
          role="alert"
          className="flex flex-col items-start gap-3 rounded-3xl border-2 border-danger bg-card p-8"
        >
          <b className="title-display text-2xl">
            {notFound ? 'Такой встречи нет' : 'Не удалось загрузить встречу'}
          </b>
          <span>
            {notFound
              ? 'Возможно, её удалили или ссылка неточная.'
              : 'Попробуйте ещё раз или вернитесь к каталогу.'}
          </span>
          <div className="flex flex-wrap gap-2.5">
            {!notFound && (
              <Button size="lg" onClick={() => void refetch()}>
                Повторить
              </Button>
            )}
            <Button
              variant="outline"
              size="lg"
              render={<Link to={STATIC_ROUTES.EVENTS} />}
            >
              Все встречи
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      <EventHero event={event} />
      <EventDetails event={event} onRefresh={() => void refetch()} />
    </>
  );
};
