import { useState } from 'react';
import { Link } from 'react-router';

import {
  formatDayShort,
  formatTime,
  type MyEntry,
  useMyEvents,
  useMyParticipations,
} from '@/entities/event';
import { SportCoverArt } from '@/entities/sport';
import { useCurrentUser } from '@/entities/user';

import { DYNAMIC_ROUTES, STATIC_ROUTES } from '@/shared/constants';
import {
  Badge,
  Button,
  Eyebrow,
  HeroBand,
  Skeleton,
  Tabs,
  TabsList,
  TabsTrigger,
} from '@/shared/ui';

type Tab = 'part' | 'org';

const STATUS = {
  confirmed: <Badge variant="success">✓ Вы записаны</Badge>,
  pending: <Badge variant="warning">◷ Ожидает решения</Badge>,
  waitlist: <Badge variant="warning">◷ Лист ожидания</Badge>,
  rejected: <Badge variant="destructive">✕ Отклонено</Badge>,
  cancelled: <Badge variant="neutral">⊘ Отменено</Badge>,
} as const;

const EntryRow = ({ entry }: { entry: MyEntry }) => {
  const e = entry.event;
  const withdrawn = e.status !== 'published';
  return (
    <Link
      to={DYNAMIC_ROUTES.EVENT(e.id)}
      className="grid grid-cols-[72px_minmax(0,1fr)_auto] items-center gap-[18px] rounded-3xl bg-card p-5 text-card-foreground max-sm:grid-cols-[56px_minmax(0,1fr)]"
    >
      <SportCoverArt
        sport={e.sport}
        rings={[]}
        glyphSize={34}
        className="size-[72px] rounded-[20px] max-sm:size-14"
      />
      <span className="flex min-w-0 flex-col gap-1">
        <b className="text-xl">{e.title}</b>
        <span className="text-ink-soft dark:text-muted-foreground">
          {formatDayShort(e.startsAt)}, {formatTime(e.startsAt)} ·{' '}
          {e.location}
        </span>
        {entry.status === 'rejected' && entry.decisionReason && (
          <span className="text-[15px] text-danger">
            Причина: {entry.decisionReason}
          </span>
        )}
        {withdrawn && e.moderationReason && (
          <span className="text-[15px]">Причина: {e.moderationReason}</span>
        )}
      </span>
      <span className="max-sm:col-span-2">
        {withdrawn ? (
          <Badge variant="neutral">⊘ Снята</Badge>
        ) : (
          STATUS[entry.status]
        )}
      </span>
    </Link>
  );
};

export const MySportPage = () => {
  const [tab, setTab] = useState<Tab>('part');
  const { data: user } = useCurrentUser();
  const part = useMyParticipations();
  const mine = useMyEvents();
  const active = tab === 'part' ? part : mine;

  const entries = (part.data ?? []).filter((x) => x.status !== 'cancelled');
  const next = entries.find(
    (x) =>
      x.status === 'confirmed' &&
      x.event.status === 'published' &&
      new Date(x.event.startsAt) > new Date(),
  );

  const renderList = () => {
    if (active.isPending)
      return (
        <>
          <Skeleton className="h-28 rounded-3xl" />
          <Skeleton className="h-28 rounded-3xl" />
        </>
      );
    if (active.isError)
      return (
        <div
          role="alert"
          className="flex flex-wrap items-center justify-between gap-4 rounded-3xl border-2 border-danger bg-card p-6"
        >
          <b>Не удалось загрузить список.</b>
          <Button onClick={() => void active.refetch()}>Повторить</Button>
        </div>
      );
    if (tab === 'part') {
      if (!entries.length)
        return (
          <div className="flex flex-col items-start gap-3 rounded-3xl bg-card p-8">
            <b className="title-display text-2xl">Пока нет записей</b>
            <span>Найдите встречу рядом — запись займёт минуту.</span>
            <Button size="lg" render={<Link to={STATIC_ROUTES.EVENTS} />}>
              Найти встречу
            </Button>
          </div>
        );
      return entries.map((x) => <EntryRow key={x.id} entry={x} />);
    }
    if (!mine.data?.items.length)
      return (
        <div className="flex flex-col items-start gap-3 rounded-3xl bg-card p-8">
          <b className="title-display text-2xl">
            Вы пока ничего не организуете
          </b>
          <Button size="lg" render={<Link to={STATIC_ROUTES.EVENT_NEW} />}>
            Создать встречу
          </Button>
        </div>
      );
    return mine.data.items.map((e) => (
      <div
        key={e.id}
        className="flex flex-wrap items-center justify-between gap-4 rounded-3xl bg-card p-[22px]"
      >
        <div className="flex min-w-0 flex-col gap-1.5">
          <div className="flex flex-wrap gap-1.5">
            <Badge variant="soft">{e.sport.name}</Badge>
            {e.status !== 'published' && (
              <Badge variant="neutral">⊘ Снята</Badge>
            )}
          </div>
          <b className="text-xl">{e.title}</b>
          <span className="text-ink-soft dark:text-muted-foreground">
            {formatDayShort(e.startsAt)}, {formatTime(e.startsAt)} · в составе{' '}
            {e.confirmedCount}
            {e.pendingCount > 0 && ` · заявок: ${e.pendingCount}`}
          </span>
        </div>
        <Button render={<Link to={DYNAMIC_ROUTES.EVENT(e.id)} />}>
          Открыть
        </Button>
      </div>
    ));
  };

  return (
    <>
      <HeroBand
        tone="blue"
        eyebrow={user?.name ? `${user.name} · мой спорт` : 'мой спорт'}
        title="Мой спорт"
      >
        <Tabs value={tab} onValueChange={(v) => setTab(v as Tab)}>
          <TabsList aria-label="Разделы" className="self-start bg-white/15">
            <TabsTrigger
              value="part"
              className="text-white data-active:text-ink data-active:bg-white dark:data-active:bg-white"
            >
              Участвую · {entries.length}
            </TabsTrigger>
            <TabsTrigger
              value="org"
              className="text-white data-active:text-ink data-active:bg-white dark:data-active:bg-white"
            >
              Организую · {mine.data?.total ?? 0}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </HeroBand>

      <div className="mx-auto flex w-full max-w-[1280px] flex-wrap items-start gap-7 px-4 pt-7 pb-18 sm:px-8">
        <div className="flex min-w-0 flex-[999_1_560px] flex-col gap-3.5">
          {renderList()}
        </div>

        <aside className="flex min-w-0 flex-[1_1_320px] flex-col gap-3.5">
          {next && (
            <div className="flex flex-col gap-2.5 rounded-[28px] bg-ink p-6 text-paper">
              <Eyebrow className="text-gold">ближайшая встреча</Eyebrow>
              <b className="title-display text-2xl">
                {formatDayShort(next.event.startsAt)},{' '}
                {formatTime(next.event.startsAt)}
              </b>
              <span>
                {next.event.title} · {next.event.location}
              </span>
              <Button
                variant="gold"
                className="self-start"
                render={<Link to={DYNAMIC_ROUTES.EVENT(next.event.id)} />}
              >
                Открыть
              </Button>
            </div>
          )}
        </aside>
      </div>
    </>
  );
};
