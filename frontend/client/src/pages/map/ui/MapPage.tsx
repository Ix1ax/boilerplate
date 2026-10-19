import { useState } from 'react';
import { Link } from 'react-router';

import { useEvents } from '@/entities/event';
import { formatWhen, JOIN_MODE_LABELS } from '@/entities/event';
import { getSportVisual } from '@/entities/sport';

import { DYNAMIC_ROUTES } from '@/shared/constants';
import { Badge, Eyebrow, HeroBand, Skeleton } from '@/shared/ui';

// Демо-город: приблизительная рамка Красноярска по координатам фикстур.
const BOUNDS = { minLat: 55.95, maxLat: 56.08, minLng: 92.68, maxLng: 93.02 };

const clamp = (value: number) => Math.min(100, Math.max(0, value));

const project = (lat: number, lng: number) => ({
  left: clamp(
    ((lng - BOUNDS.minLng) / (BOUNDS.maxLng - BOUNDS.minLng)) * 100,
  ),
  top: clamp(
    ((BOUNDS.maxLat - lat) / (BOUNDS.maxLat - BOUNDS.minLat)) * 100,
  ),
});

export const MapPage = () => {
  const { data, isPending } = useEvents({ limit: 100 });
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const located = (data?.items ?? []).filter(
    (event): event is typeof event & { lat: number; lng: number } =>
      event.lat != null && event.lng != null,
  );
  const selected = located.find((event) => event.id === selectedId);

  return (
    <>
      <HeroBand
        eyebrow="Красноярск · демо-город"
        title="Встречи на карте"
        description="Схема без реальных тайлов: точки расставлены по координатам встреч внутри города."
      />
      <div className="mx-auto w-full max-w-[1280px] px-4 pt-7 pb-18 sm:px-8">
        {isPending && <Skeleton className="h-[560px] w-full rounded-3xl" />}

        {!isPending && (
          <div className="grid gap-6 lg:grid-cols-[1fr_360px]">
            <div
              className="relative aspect-square overflow-hidden rounded-3xl border border-border bg-[linear-gradient(rgb(18_18_18/6%)_1px,transparent_1px),linear-gradient(90deg,rgb(18_18_18/6%)_1px,transparent_1px)] bg-card bg-[size:48px_48px] dark:bg-[linear-gradient(rgb(238_240_234/8%)_1px,transparent_1px),linear-gradient(90deg,rgb(238_240_234/8%)_1px,transparent_1px)]"
              role="img"
              aria-label="Схема расположения встреч в городе"
            >
              {located.map((event) => {
                const { left, top } = project(event.lat, event.lng);
                const visual = getSportVisual(event.sport.name);
                const isSelected = event.id === selectedId;
                return (
                  <button
                    key={event.id}
                    type="button"
                    aria-label={event.title}
                    aria-pressed={isSelected}
                    onClick={() => setSelectedId(event.id)}
                    className="absolute flex -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full border-2 border-white shadow-md transition-transform"
                    style={{
                      left: `${left}%`,
                      top: `${top}%`,
                      width: isSelected ? 22 : 14,
                      height: isSelected ? 22 : 14,
                      background: visual.cover.ink,
                      zIndex: isSelected ? 2 : 1,
                    }}
                  />
                );
              })}
              {located.length === 0 && (
                <p className="absolute inset-0 grid place-items-center px-6 text-center text-muted-foreground">
                  У ближайших встреч пока не указаны координаты.
                </p>
              )}
            </div>

            <div className="flex max-h-[560px] flex-col gap-3 overflow-y-auto">
              {located.length === 0 && !isPending && (
                <p className="text-muted-foreground">Список пуст.</p>
              )}
              {located.map((event) => (
                <button
                  key={event.id}
                  type="button"
                  onClick={() => setSelectedId(event.id)}
                  className={`flex flex-col gap-1.5 rounded-2xl p-4 text-left transition-colors ${
                    event.id === selectedId
                      ? 'bg-ink text-paper'
                      : 'bg-card hover:bg-muted'
                  }`}
                >
                  <Eyebrow
                    className={
                      event.id === selectedId
                        ? 'text-gold'
                        : 'text-muted-foreground'
                    }
                  >
                    {formatWhen(event.starts_at)}
                  </Eyebrow>
                  <b>{event.title}</b>
                  <span className="text-sm opacity-80">{event.location}</span>
                  <div className="mt-1 flex flex-wrap gap-1.5">
                    <Badge variant={event.id === selectedId ? 'gold' : 'soft'}>
                      {event.sport.name}
                    </Badge>
                    <Badge variant="outline">
                      {JOIN_MODE_LABELS[event.join_mode]}
                    </Badge>
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        {selected && (
          <Link
            to={DYNAMIC_ROUTES.EVENT(selected.id)}
            className="mt-6 inline-flex min-h-12 items-center rounded-full bg-primary px-6 font-bold text-primary-foreground"
          >
            Открыть «{selected.title}»
          </Link>
        )}
      </div>
    </>
  );
};
