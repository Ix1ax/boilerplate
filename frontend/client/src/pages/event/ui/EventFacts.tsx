import { Link } from 'react-router';

import {
  type Event,
  formatDayLong,
  formatTeamCount,
  formatTime,
  LEVEL_LABELS,
} from '@/entities/event';

import { DYNAMIC_ROUTES } from '@/shared/constants';
import { initialOf } from '@/shared/lib';

const Tile = ({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) => (
  <div className="flex flex-col gap-1 rounded-[22px] bg-card px-5 py-[18px]">
    <span className="font-mono text-[13px] text-muted-foreground uppercase">
      {label}
    </span>
    {children}
  </div>
);

const osmLink = (lat: number, lng: number) =>
  `https://www.openstreetmap.org/?mlat=${lat}&mlon=${lng}#map=17/${lat}/${lng}`;

/** Четыре плитки «главного»: когда, где, для кого, кто организует. */
export const EventFacts = ({ event }: { event: Event }) => {
  const orgLink =
    event.organizer.type === 'organization'
      ? DYNAMIC_ROUTES.ORGANIZATION(event.organizer.id)
      : DYNAMIC_ROUTES.PERSON(event.organizer.id);

  return (
    <section
      aria-label="Главное о встрече"
      className="grid grid-cols-[repeat(auto-fit,minmax(min(230px,100%),1fr))] gap-3"
    >
      <Tile label="Когда">
        <b className="text-xl leading-snug">{formatDayLong(event.startsAt)}</b>
        <span>
          начало в {formatTime(event.startsAt)}
          {event.endsAt && `, до ${formatTime(event.endsAt)}`} · по Красноярску
        </span>
      </Tile>
      <Tile label="Где">
        <b className="text-xl leading-snug">{event.location}</b>
        {event.lat != null && event.lng != null && (
          <a
            href={osmLink(event.lat, event.lng)}
            target="_blank"
            rel="noreferrer"
            className="font-bold text-primary hover:text-blue-hover dark:text-interest"
          >
            на карте OpenStreetMap
          </a>
        )}
      </Tile>
      <Tile label="Для кого">
        <b className="text-xl leading-snug">{LEVEL_LABELS[event.level]}</b>
        <span>
          {formatTeamCount(event)}
          {event.firstTimersCount > 0 &&
            ` · впервые: ${event.firstTimersCount}`}
        </span>
      </Tile>
      <Link
        to={orgLink}
        className="flex items-center gap-3 rounded-[22px] bg-card px-5 py-[18px] text-card-foreground"
      >
        <span
          aria-hidden="true"
          className="grid size-12 shrink-0 place-items-center rounded-full bg-blue text-[19px] font-extrabold text-white"
        >
          {initialOf(event.organizer.name)}
        </span>
        <span className="flex min-w-0 flex-col">
          <span className="font-mono text-[13px] text-muted-foreground uppercase">
            Организует
          </span>
          <b className="text-[19px] leading-snug underline underline-offset-3">
            {event.organizer.name}
          </b>
          <span className="text-[15px]">
            {event.organizer.type === 'organization'
              ? 'организация · допущена'
              : 'человек'}
          </span>
        </span>
      </Link>
    </section>
  );
};
