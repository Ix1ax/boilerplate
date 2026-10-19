import { dayPartLabels } from '@starter/contracts';

import { type Sport, useDemand } from '@/entities/sport';

import { Skeleton } from '@/shared/ui';

interface SportDemandCardProps {
  sport: Sport;
}

const DAY_PARTS = Object.keys(dayPartLabels) as (keyof typeof dayPartLabels)[];

/** Обезличенный спрос на одно направление: сколько человек ищут компанию и когда. */
export const SportDemandCard = ({ sport }: SportDemandCardProps) => {
  const { data, isPending } = useDemand({ sportId: sport.id });

  if (isPending) return <Skeleton className="h-28 w-full rounded-2xl" />;
  if (!data || data.count === 0) return null;

  const max = Math.max(...DAY_PARTS.map((part) => data.byDayPart[part]), 1);

  return (
    <div className="flex flex-col gap-3 rounded-2xl bg-card p-5">
      <div className="flex items-baseline justify-between gap-2">
        <b className="text-lg">{sport.name}</b>
        <span className="text-sm text-muted-foreground">
          {data.count} {data.count === 1 ? 'человек ищет' : 'человек ищут'}
        </span>
      </div>
      <div className="flex items-end gap-3">
        {DAY_PARTS.map((part) => (
          <div key={part} className="flex flex-1 flex-col items-center gap-1">
            <div className="flex h-16 w-full items-end">
              <div
                className="w-full rounded-t-md bg-blue"
                style={{
                  height: `${(data.byDayPart[part] / max) * 100}%`,
                }}
              />
            </div>
            <span className="text-xs text-muted-foreground">
              {dayPartLabels[part]}
            </span>
            <span className="text-sm font-semibold">
              {data.byDayPart[part]}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
};
