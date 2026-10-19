import { Link } from 'react-router';
import { cn } from 'cn';

import { SportCoverArt } from '@/entities/sport';

import { DYNAMIC_ROUTES } from '@/shared/constants';
import { Lift } from '@/shared/motion';
import { Badge } from '@/shared/ui';

import {
  formatTeamCount,
  formatWhen,
  JOIN_MODE_LABELS,
  levelShort,
  organizerKind,
} from '../lib';
import type { Event } from '../model';

const MetaLabel = ({ children }: { children: string }) => (
  <div className="font-mono text-xs text-muted-foreground uppercase">
    {children}
  </div>
);

interface EventCardProps {
  event: Event;
  className?: string;
}

/** Карточка встречи: радар-обложка, для кого, где, кто организует и сколько в составе. */
export const EventCard = ({ event, className }: EventCardProps) => {
  const status = event.myParticipation?.status;
  return (
    <Lift className="flex h-full w-full rounded-3xl">
      <Link
        to={DYNAMIC_ROUTES.EVENT(event.id)}
        className={cn(
          'flex w-full flex-col overflow-hidden rounded-3xl bg-card text-card-foreground no-underline shadow-[0_1px_2px_rgb(18_18_18/6%),0_8px_24px_-14px_rgb(18_18_18/20%)]',
          className,
        )}
      >
        <SportCoverArt sport={event.sport.name} className="h-[170px] shrink-0">
          <span className="absolute top-3.5 left-3.5 rounded-full bg-ink px-3 py-1.5 text-sm font-semibold text-paper">
            {formatWhen(event.startsAt)}
          </span>
          <span
            className={cn(
              'absolute right-3.5 bottom-3.5 rounded-full border-[1.5px] border-ink bg-white px-3 py-1.5 text-sm font-semibold text-ink',
              event.joinMode === 'request' && 'border-dashed',
            )}
          >
            {JOIN_MODE_LABELS[event.joinMode]}
          </span>
        </SportCoverArt>
        <div className="flex flex-1 flex-col gap-2.5 px-[22px] pt-5 pb-[22px]">
          <div className="flex flex-wrap gap-1.5">
            <Badge variant="soft">{event.sport.name}</Badge>
            {event.sport.isCustom && (
              <Badge variant="gold">Новое направление</Badge>
            )}
            {event.isOrganizer && (
              <Badge variant="outline">Вы организатор</Badge>
            )}
            {status === 'confirmed' && (
              <Badge variant="success">✓ Вы записаны</Badge>
            )}
            {status === 'pending' && (
              <Badge variant="warning">◷ Ожидает решения</Badge>
            )}
            {status === 'waitlist' && (
              <Badge variant="warning">◷ Лист ожидания</Badge>
            )}
          </div>
          <b className="text-[22px] leading-tight">{event.title}</b>
          <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-[15px] leading-snug">
            <div className="min-w-0">
              <MetaLabel>Где</MetaLabel>
              <span className="line-clamp-2">{event.location}</span>
            </div>
            <div>
              <MetaLabel>Для кого</MetaLabel>
              {levelShort(event.level)}
            </div>
          </div>
          <div className="mt-auto flex flex-wrap justify-between gap-1.5 border-t border-stone pt-3 text-[15px] dark:border-border">
            <span>
              <b>{event.organizer.name}</b>{' '}
              <span className="text-muted-foreground">
                · {organizerKind(event.organizer)}
              </span>
            </span>
            <span className="text-muted-foreground">
              {formatTeamCount(event)}
            </span>
          </div>
        </div>
      </Link>
    </Lift>
  );
};

/** Заглушка карточки на время загрузки. */
export const EventCardSkeleton = () => (
  <div className="overflow-hidden rounded-3xl bg-card" aria-hidden="true">
    <div className="h-[170px] animate-pulse bg-stone motion-reduce:animate-none dark:bg-muted" />
    <div className="flex flex-col gap-3 p-[22px]">
      <div className="h-4 w-2/5 rounded-lg bg-stone dark:bg-muted" />
      <div className="h-[26px] w-[85%] rounded-lg bg-stone dark:bg-muted" />
      <div className="h-4 w-3/5 rounded-lg bg-stone dark:bg-muted" />
    </div>
  </div>
);
