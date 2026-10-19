import type { Sport } from '@/entities/sport';

import { Eyebrow } from '@/shared/ui';

import { SportDemandCard } from './SportDemandCard';

interface ClubDemandPanelProps {
  sports: Sport[];
}

/** «Спрос рядом»: сколько людей обезличенно ищут компанию по направлениям клуба. */
export const ClubDemandPanel = ({ sports }: ClubDemandPanelProps) => {
  if (sports.length === 0) return null;

  return (
    <section className="flex flex-col gap-4">
      <div>
        <Eyebrow className="text-blue">спрос рядом</Eyebrow>
        <h2 className="title-display mt-1.5 text-2xl">
          Сколько людей ищут ваш спорт
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Без имён и контактов. Если спрос есть — стоит опубликовать встречу.
        </p>
      </div>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(280px,100%),1fr))] gap-4">
        {sports.map((sport) => (
          <SportDemandCard key={sport.id} sport={sport} />
        ))}
      </div>
    </section>
  );
};
