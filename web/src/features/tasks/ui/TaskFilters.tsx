import { useSearchParams } from 'react-router';

import { STATUS_META, STATUS_ORDER, type TaskStatus } from '@/entities/task';

import {
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui';

const SORTS = [
  { value: '-created', label: 'Сначала новые' },
  { value: 'created', label: 'Сначала старые' },
  { value: 'title', label: 'По названию А→Я' },
  { value: 'status', label: 'По статусу' },
] as const;

export const TaskFilters = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const q = searchParams.get('q') ?? '';
  const status = searchParams.get('status') ?? '';
  const sort = searchParams.get('sort') ?? '-created';

  const update = (key: string, value: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (value) next.set(key, value);
      else next.delete(key);
      next.delete('offset');
      return next;
    });
  };

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Input
        value={q}
        onChange={(event) => update('q', event.target.value)}
        placeholder="Поиск по названию…"
        aria-label="Поиск задач"
        className="max-w-xs"
      />
      <Select
        value={status || 'all'}
        onValueChange={(value) =>
          update('status', value === 'all' ? '' : (value ?? ''))
        }
      >
        <SelectTrigger aria-label="Фильтр по статусу">
          <SelectValue placeholder="Все статусы" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Все статусы</SelectItem>
          {STATUS_ORDER.map((value: TaskStatus) => (
            <SelectItem key={value} value={value}>
              {STATUS_META[value].label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select
        value={sort}
        onValueChange={(value) => update('sort', value ?? '-created')}
      >
        <SelectTrigger aria-label="Сортировка">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {SORTS.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};
