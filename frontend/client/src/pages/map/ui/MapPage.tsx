import { useState } from 'react';
import { Link } from 'react-router';

import { formatWhen, useEventPages } from '@/entities/event';
import { useSports } from '@/entities/sport';

import { DYNAMIC_ROUTES } from '@/shared/constants';
import { useDebounce } from '@/shared/hooks';
import { Badge, Button, HeroBand, Notice, Skeleton } from '@/shared/ui';

import { EventMap } from './EventMap';

export const MapPage = () => {
  const [bounds, setBounds] = useState('92.68,55.95,93.02,56.08');
  const [sport, setSport] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [center, setCenter] = useState<[number, number] | null>(null);
  const [geoError, setGeoError] = useState('');
  const [locating, setLocating] = useState(false);
  const bbox = useDebounce(bounds, 300);
  const query = useEventPages({ bbox, sport: sport || undefined });
  const { data: sports = [] } = useSports();
  const located =
    query.data?.pages
      .flatMap((page) => page.items)
      .filter((event) => event.lat != null && event.lng != null) ?? [];
  const selected = located.find((event) => event.id === selectedId);
  const atPlace = selected
    ? located.filter(
        (event) =>
          event.location === selected.location &&
          event.lat?.toFixed(4) === selected.lat?.toFixed(4) &&
          event.lng?.toFixed(4) === selected.lng?.toFixed(4),
      )
    : located;
  const locate = () => {
    if (!navigator.geolocation) {
      setGeoError(
        'Браузер не поддерживает определение местоположения. Выберите область на карте.',
      );
      return;
    }
    setLocating(true);
    setGeoError('');
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        setCenter([coords.latitude, coords.longitude]);
        setLocating(false);
      },
      () => {
        setLocating(false);
        setGeoError(
          'Не удалось определить местоположение. Разрешите доступ в браузере или выберите область вручную.',
        );
      },
      { timeout: 10000, maximumAge: 60000 },
    );
  };
  return (
    <>
      <HeroBand
        eyebrow="Красноярск · встречи рядом"
        title="Встречи на карте"
        description="Выберите площадку, чтобы увидеть встречи. Число на метке — количество загруженных встреч в этом месте."
      />
      <div className="mx-auto grid w-full max-w-[1280px] gap-5 px-4 pt-7 pb-18 sm:px-8">
        <div className="flex flex-wrap items-end gap-3">
          <div className="grid gap-1.5">
            <label htmlFor="map-sport">Направление</label>
            <select
              id="map-sport"
              className="min-h-12 rounded-lg border border-input bg-card px-3"
              value={sport}
              onChange={(e) => {
                setSport(e.target.value);
                setSelectedId(null);
              }}
            >
              <option value="">Все направления</option>
              {sports.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
          </div>
          <Button variant="outline" disabled={locating} onClick={locate}>
            {locating ? 'Определяем…' : 'Показать рядом со мной'}
          </Button>
          <Button variant="outline" onClick={() => setSelectedId(null)}>
            Все площадки
          </Button>
        </div>
        {geoError && <Notice role="alert">{geoError}</Notice>}
        {query.isError && (
          <Notice tone="danger" role="alert">
            Не удалось загрузить встречи.{' '}
            <Button variant="link" onClick={() => void query.refetch()}>
              Повторить
            </Button>
          </Notice>
        )}
        <div className="grid gap-6 lg:grid-cols-[1fr_360px]">
          <EventMap
            events={located}
            selected={selectedId}
            onSelect={setSelectedId}
            onBounds={setBounds}
            center={center}
          />
          <section
            className="flex max-h-[520px] flex-col gap-3 overflow-y-auto"
            aria-label="Встречи на выбранных площадках"
          >
            <h2 className="title-display text-2xl">
              {selected ? selected.location : 'В видимой области'}
            </h2>
            {query.isPending && <Skeleton className="h-40 rounded-3xl" />}
            {!query.isPending && !query.isError && !atPlace.length && (
              <p>
                В этой области встреч пока нет. Измените масштаб или
                направление.
              </p>
            )}
            {atPlace.map((event) => (
              <article
                key={event.id}
                className="grid gap-2 rounded-3xl bg-card p-5"
              >
                <span>{formatWhen(event.startsAt)}</span>
                <h3 className="font-bold">{event.title}</h3>
                <p>{event.location}</p>
                <Badge>{event.sport.name}</Badge>
                <Button
                  variant="outline"
                  render={<Link to={DYNAMIC_ROUTES.EVENT(event.id)} />}
                >
                  Открыть встречу
                </Button>
              </article>
            ))}
            {query.hasNextPage && (
              <Button
                disabled={query.isFetchingNextPage}
                onClick={() => void query.fetchNextPage()}
              >
                {query.isFetchingNextPage
                  ? 'Загружаем…'
                  : 'Загрузить ещё встречи в области'}
              </Button>
            )}
          </section>
        </div>
        <p className="text-sm text-muted-foreground">
          Картографические данные © OpenStreetMap. Встречи без координат
          доступны в каталоге. Ваше местоположение используется только после
          нажатия кнопки.
        </p>
      </div>
    </>
  );
};
