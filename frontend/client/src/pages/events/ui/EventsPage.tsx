import { Link } from 'react-router';

import {
  describeFilters,
  EventFiltersPanel,
  useEventFilters,
} from '@/features/event-filters';

import { EventCard, EventCardSkeleton, useEventPages } from '@/entities/event';
import { useSports } from '@/entities/sport';

import { STATIC_ROUTES } from '@/shared/constants';
import { Lift, Reveal, staggerDelay } from '@/shared/motion';
import { Button, HeroBand, RadarRings } from '@/shared/ui';

const GRID =
  'grid grid-cols-[repeat(auto-fill,minmax(min(320px,100%),1fr))] gap-5';

/** Синяя карточка в конце выдачи: если подходящего нет — собрать компанию. */
const CompanyTeaser = () => (
  <Lift className="flex h-full w-full rounded-3xl">
    <Link
      to={STATIC_ROUTES.COMPANY}
      className="relative flex min-h-[300px] w-full flex-col justify-between gap-4 overflow-hidden rounded-3xl bg-blue p-7 text-white"
    >
      <RadarRings
        tone="white"
        sizes={[140, 240]}
        strength={0.3}
        className="right-[50px] bottom-[50px]"
      />
      <b className="display-caps relative text-[26px] leading-[1.05]">
        Не нашли своё?
      </b>
      <span className="relative">
        Оставьте запрос: спорт, район и удобное время. Когда найдётся подходящая
        встреча, мы сообщим.
      </span>
      <span className="relative inline-flex min-h-12 items-center self-start rounded-full bg-gold px-5 font-bold text-ink">
        Собрать компанию
      </span>
    </Link>
  </Lift>
);

export const EventsPage = () => {
  const { filters, setFilter, reset, params } = useEventFilters();
  const {
    data,
    isPending,
    isError,
    isFetching,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchNextPageError,
  } = useEventPages(params);
  const { data: sports } = useSports();

  const items = data?.pages.flatMap((page) => page.items) ?? [];
  const total = data?.pages[0]?.total ?? 0;
  let moreLabel = 'Показать ещё';
  if(isFetchNextPageError) moreLabel = 'Повторить загрузку следующей страницы';
  if(isFetching) moreLabel = 'Загружаем…';

  return (
    <>
      <HeroBand
        overlap
        eyebrow="Красноярск · встречи рядом"
        title={
          <>
            Найди своих <span className="text-gold">рядом</span>
          </>
        }
        description="В каждой карточке видно, кто организует, для кого встреча и как попасть в состав."
      />

      <div className="relative mx-auto -mt-[72px] flex w-full max-w-[1280px] flex-col gap-7 px-4 pb-18 sm:px-8">
        <EventFiltersPanel
          filters={filters}
          onChange={(key, value) => {
            setFilter(key, value);
          }}
        />

        {isPending && (
          <div aria-busy="true" aria-label="Загружаем встречи" className={GRID}>
            <EventCardSkeleton />
            <EventCardSkeleton />
            <EventCardSkeleton />
          </div>
        )}

        {isError && !data && (
          <div
            role="alert"
            className="flex flex-wrap items-center justify-between gap-5 rounded-3xl border-2 border-danger bg-card p-8"
          >
            <div className="flex flex-col gap-1.5">
              <b className="title-display text-2xl">
                Не удалось загрузить встречи
              </b>
              <span>
                Проверьте соединение и попробуйте ещё раз. Фильтры сохранятся.
              </span>
            </div>
            <Button size="lg" onClick={() => void refetch()}>
              Повторить
            </Button>
          </div>
        )}

        {!isPending && !(isError && !data) && (
          <div className="flex flex-col gap-[18px]">
            <div
              aria-live="polite"
              className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-2"
            >
              <h2 className="title-display text-3xl">Ближайшие встречи</h2>
              <span className="font-semibold text-muted-foreground">
                Найдено: {total} · сначала ближайшие
              </span>
            </div>

            {items.length === 0 ? (
              <div className="grid grid-cols-[repeat(auto-fit,minmax(min(420px,100%),1fr))] gap-5">
                <div className="relative flex flex-col items-start gap-3.5 overflow-hidden rounded-3xl bg-card p-8">
                  <RadarRings
                    tone="blue"
                    sizes={[100, 200]}
                    strength={0.32}
                    className="top-10 right-10"
                  />
                  <b className="title-display relative text-2xl">
                    По этим условиям встреч пока нет
                  </b>
                  <span className="relative">
                    Ваши условия: <b>{describeFilters(filters, sports)}</b>. Они
                    сохранены — можно изменить одно или сбросить все.
                  </span>
                  <Button variant="outline" size="lg" onClick={reset}>
                    Сбросить фильтры
                  </Button>
                </div>
                <div className="flex flex-col items-start gap-3.5 rounded-3xl bg-ink p-8 text-paper">
                  <b className="title-display text-2xl text-gold">
                    Соберите компанию с нуля
                  </b>
                  <span>
                    Укажите спорт, район и удобное время. Когда кто-то создаст
                    подходящую встречу, мы сообщим. Это не запись — записаться
                    можно будет отдельно.
                  </span>
                  <Button
                    variant="gold"
                    size="lg"
                    render={<Link to={STATIC_ROUTES.COMPANY} />}
                  >
                    Собрать компанию
                  </Button>
                  <span>
                    или{' '}
                    <Link
                      to={STATIC_ROUTES.EVENT_NEW}
                      className="font-bold text-gold underline underline-offset-3"
                    >
                      создайте свою встречу
                    </Link>
                  </span>
                </div>
              </div>
            ) : (
              <div className={GRID} aria-busy={isFetching}>
                {items.map((event, index) => (
                  <Reveal
                    key={event.id}
                    className="h-full"
                    delay={staggerDelay(index)}
                    y={20}
                  >
                    <EventCard event={event} className="h-full" />
                  </Reveal>
                ))}
                <Reveal
                  className="h-full"
                  delay={staggerDelay(items.length)}
                  y={20}
                >
                  <CompanyTeaser />
                </Reveal>
              </div>
            )}

            {hasNextPage && (
              <Button
                variant="outline"
                size="lg"
                className="self-center"
                disabled={isFetching}
                onClick={() => void fetchNextPage()}
              >
                {moreLabel}
              </Button>
            )}
          </div>
        )}
      </div>
    </>
  );
};
