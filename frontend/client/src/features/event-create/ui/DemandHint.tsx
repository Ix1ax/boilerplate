import { dayPartLabels, weekdayLabels } from '@starter/contracts';

import { useDemand } from '@/entities/sport';

interface DemandHintProps {
  sportId?: string;
  sportName?: string;
}

/** «12 человек ищут компанию на бег, чаще утром в выходные» — обезличенный спрос под выбором направления. */
export const DemandHint = ({ sportId, sportName }: DemandHintProps) => {
  const { data, isPending } = useDemand({ sportId, sportName });
  if (isPending || !data || data.count === 0) return null;

  const dayPartEntries = Object.entries(data.byDayPart) as [
    keyof typeof dayPartLabels,
    number,
  ][];
  const busiestPart = dayPartEntries.reduce((a, b) => (b[1] > a[1] ? b : a));
  const busiestWeekday = data.byWeekday.reduce(
    (best, count, index) => (count > best.count ? { index, count } : best),
    { index: 0, count: 0 },
  );

  return (
    <p className="rounded-2xl bg-blue-soft px-4 py-3 text-sm text-blue">
      {data.count}{' '}
      {data.count === 1 ? 'человек ищет' : 'человек ищут'} компанию на это
      направление
      {busiestPart[1] > 0 && `, чаще ${dayPartLabels[busiestPart[0]].toLowerCase()}`}
      {busiestWeekday.count > 0 &&
        ` (особенно в ${weekdayLabels[busiestWeekday.index]})`}
      .
    </p>
  );
};
