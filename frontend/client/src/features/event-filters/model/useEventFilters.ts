import { useMemo } from 'react';
import { useSearchParams } from 'react-router';

import {
  DAY_RANGE_LABELS,
  type DayRange,
  dayRangeParams,
  type EventListParams,
} from '@/entities/event';
import type { Sport } from '@/entities/sport';

import { useDebounce } from '@/shared/hooks';

export type EventFilters = {
  zone: string;
  stepFree: boolean;
  q: string;
  sport: string;
  day: DayRange;
  mode: '' | 'open' | 'request';
  novice: boolean;
};

const DEFAULTS: EventFilters = {
  zone: '',
  stepFree: false,
  q: '',
  sport: '',
  day: 'week',
  mode: '',
  novice: false,
};

const isDayRange = (value: string | null): value is DayRange =>
  !!value && value in DAY_RANGE_LABELS;

/** Фильтры каталога живут в адресе: ссылкой можно поделиться, «Назад» их сохраняет. */
export const useEventFilters = () => {
  const [search, setSearch] = useSearchParams();

  const filters: EventFilters = {
    zone: search.get('zone') ?? '',
    stepFree: search.get('stepFree') === '1',
    q: search.get('q') ?? DEFAULTS.q,
    sport: search.get('sport') ?? DEFAULTS.sport,
    day: isDayRange(search.get('day'))
      ? (search.get('day') as DayRange)
      : DEFAULTS.day,
    mode:
      (['open', 'request'] as const).find((m) => m === search.get('mode')) ??
      '',
    novice: search.get('novice') === '1',
  };

  const setFilter = <K extends keyof EventFilters>(
    key: K,
    value: EventFilters[K],
  ) => {
    setSearch(
      (prev) => {
        const next = new URLSearchParams(prev);
        let raw = String(value);
        if (typeof value === 'boolean') raw = value ? '1' : '';
        if (!raw || raw === String(DEFAULTS[key])) next.delete(key);
        else next.set(key, raw);
        return next;
      },
      { replace: true },
    );
  };

  const reset = () => setSearch({}, { replace: true });

  const q = useDebounce(filters.q.trim(), 300);
  const params = useMemo<EventListParams>(
    () => ({
      zone: filters.zone || undefined,
      step_free: filters.stepFree || undefined,
      q: q || undefined,
      sport: filters.sport || undefined,
      mode: filters.mode || undefined,
      level: filters.novice ? 'beginner' : undefined,
      ...dayRangeParams(filters.day),
    }),
    [
      q,
      filters.zone,
      filters.stepFree,
      filters.sport,
      filters.mode,
      filters.novice,
      filters.day,
    ],
  );

  const isDefault =
    !filters.zone &&
    !filters.stepFree &&
    !filters.q &&
    !filters.sport &&
    filters.day === DEFAULTS.day &&
    !filters.mode &&
    !filters.novice;

  return { filters, setFilter, reset, params, isDefault };
};

/** Условия словами — для пустого состояния: «Шахматы · завтра · без опыта». */
export const describeFilters = (
  filters: EventFilters,
  sports: Sport[] = [],
) => {
  const parts: string[] = [];
  if (filters.zone) parts.push(filters.zone);
  if (filters.stepFree) parts.push('доступная площадка');
  const sport = sports.find((item) => item.id === filters.sport);
  if (sport) parts.push(sport.name);
  if (filters.day !== 'all')
    parts.push(DAY_RANGE_LABELS[filters.day].toLowerCase());
  if (filters.mode === 'open') parts.push('свободная запись');
  if (filters.mode === 'request') parts.push('по заявке');
  if (filters.novice) parts.push('без опыта');
  if (filters.q.trim()) parts.push(`«${filters.q.trim()}»`);
  return parts.join(' · ') || 'без фильтров';
};
