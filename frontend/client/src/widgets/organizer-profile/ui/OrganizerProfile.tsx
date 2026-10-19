import type { ReactNode } from 'react';
import { Link } from 'react-router';

import { type Event,EventCard } from '@/entities/event';
import type { Profile } from '@/entities/profile';

import { STATIC_ROUTES } from '@/shared/constants';
import { Button, HeroBand } from '@/shared/ui';

interface OrganizerProfileProps {
  name: string;
  description: string;
  eyebrow: string;
  stats: Profile['stats'];
  events: Event[];
  children?: ReactNode;
}

export const OrganizerProfile = ({
  name,
  description,
  eyebrow,
  stats,
  events,
  children,
}: OrganizerProfileProps) => (
  <>
    <HeroBand title={name} eyebrow={eyebrow} description={description} />
    <div className="mx-auto flex w-full max-w-[1280px] flex-col gap-7 px-4 pt-7 pb-18 sm:px-8">
      {children}
      <dl className="grid grid-cols-1 gap-4 rounded-3xl bg-card p-6 sm:grid-cols-3">
        {[
          ['Проведено встреч', stats.eventsHeld],
          ['Подтверждённых записей на прошедшие встречи', stats.peopleCame],
          ['Ближайших встреч', stats.upcomingEvents],
        ].map(([label, value]) => (
          <div key={label}>
            <dt className="text-muted-foreground">{label}</dt>
            <dd className="mt-1 text-3xl font-bold">{value}</dd>
          </div>
        ))}
      </dl>
      <section aria-labelledby="organizer-events">
        <h2 id="organizer-events" className="title-display mb-5 text-[26px]">
          Ближайшие встречи
        </h2>
        {events.length ? (
          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {events.map((event) => (
              <EventCard key={event.id} event={event} />
            ))}
          </div>
        ) : (
          <div className="flex flex-col items-start gap-3 rounded-3xl bg-card p-8">
            <b className="title-display text-2xl">Пока нет встреч</b>
            <p>У организатора пока нет ближайших опубликованных встреч.</p>
            <Button size="lg" render={<Link to={STATIC_ROUTES.EVENTS} />}>
              Найти другие встречи
            </Button>
          </div>
        )}
      </section>
    </div>
  </>
);
