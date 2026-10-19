import { SearchIcon } from 'lucide-react';

import { DAY_RANGE_LABELS, type DayRange } from '@/entities/event';
import { useSports } from '@/entities/sport';

import {
  Checkbox,
  Chip,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui';

import type { EventFilters } from '../model';

const DAY_ITEMS = (Object.keys(DAY_RANGE_LABELS) as DayRange[]).map(
  (value) => ({
    value,
    label: DAY_RANGE_LABELS[value],
  }),
);

const MODE_ITEMS = [
  { value: 'any', label: 'Любая запись' },
  { value: 'open', label: 'Свободная запись' },
  { value: 'request', label: 'По заявке' },
];

// Чипов не больше восьми: остальные направления находятся поиском.
const MAX_CHIPS = 8;

interface EventFiltersPanelProps {
  filters: EventFilters;
  onChange: <K extends keyof EventFilters>(
    key: K,
    value: EventFilters[K],
  ) => void;
}

/** Белая панель фильтров каталога: поиск, дата, режим записи, чипы видов спорта. */
export const EventFiltersPanel = ({
  filters,
  onChange,
}: EventFiltersPanelProps) => {
  const { data: sports = [] } = useSports();
  const base = sports.filter((sport) => !sport.is_custom);
  const chips = [
    ...base.slice(0, MAX_CHIPS),
    // Выбранное по ссылке направление показываем, даже если оно не попало в чипы.
    ...sports.filter(
      (sport) =>
        sport.id === filters.sport &&
        !base.slice(0, MAX_CHIPS).some((item) => item.id === sport.id),
    ),
  ];

  return (
    <section
      aria-label="Фильтры"
      className="flex flex-col gap-4 rounded-3xl bg-card p-5 shadow-[0_30px_60px_-30px_rgb(18_18_18/40%)]"
    >
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex min-w-0 flex-[2_1_320px] flex-col gap-1.5">
          <label htmlFor="catalog-q" className="text-[15px] font-semibold">
            Поиск
          </label>
          <div className="relative">
            <SearchIcon
              aria-hidden
              className="pointer-events-none absolute top-1/2 left-4 size-5 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              id="catalog-q"
              type="search"
              value={filters.q}
              maxLength={100}
              onChange={(event) => onChange('q', event.target.value)}
              placeholder="Название, вид спорта или организатор"
              className="pl-12"
            />
          </div>
        </div>
        <div className="flex min-w-0 flex-[1_1_200px] flex-col gap-1.5">
          <label htmlFor="catalog-day" className="text-[15px] font-semibold">
            Дата
          </label>
          <Select
            items={DAY_ITEMS}
            value={filters.day}
            onValueChange={(value) =>
              onChange('day', (value as DayRange) ?? 'week')
            }
          >
            <SelectTrigger id="catalog-day" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {DAY_ITEMS.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex min-w-0 flex-[1_1_200px] flex-col gap-1.5">
          <label htmlFor="catalog-mode" className="text-[15px] font-semibold">
            Запись
          </label>
          <Select
            items={MODE_ITEMS}
            value={filters.mode || 'any'}
            onValueChange={(value) =>
              onChange(
                'mode',
                value === 'open' || value === 'request' ? value : '',
              )
            }
          >
            <SelectTrigger id="catalog-mode" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MODE_ITEMS.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <div
          role="group"
          aria-label="Вид спорта"
          className="flex flex-wrap gap-2"
        >
          <Chip pressed={!filters.sport} onClick={() => onChange('sport', '')}>
            Все
          </Chip>
          {chips.map((sport) => (
            <Chip
              key={sport.id}
              pressed={filters.sport === sport.id}
              onClick={() =>
                onChange('sport', filters.sport === sport.id ? '' : sport.id)
              }
            >
              {sport.name}
            </Chip>
          ))}
        </div>
        <label className="flex min-h-11 items-center gap-2.5 text-base font-semibold">
          <Checkbox
            checked={filters.novice}
            onCheckedChange={(checked) => onChange('novice', checked)}
          />
          Можно без опыта
        </label>
      </div>
    </section>
  );
};
