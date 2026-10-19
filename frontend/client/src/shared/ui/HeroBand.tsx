import type { ReactNode } from 'react';
import { cn } from 'cn';

import { Eyebrow } from './Eyebrow';
import { RadarRings } from './RadarRings';

const TONES = {
  ink: {
    section: 'bg-ink text-paper',
    eyebrow: 'text-paper/80',
    rings: 'gold',
  },
  blue: {
    section: 'bg-blue text-white',
    eyebrow: 'text-white/85',
    rings: 'white',
  },
} as const;

interface HeroBandProps {
  tone?: keyof typeof TONES;
  eyebrow?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
  className?: string;
  /** Дополнительный нижний отступ, если на секцию «наезжает» панель. */
  overlap?: boolean;
}

/** Первая секция экрана: мета-строка, крупный заголовок капсом и волны радара справа. */
export const HeroBand = ({
  tone = 'ink',
  eyebrow,
  title,
  description,
  children,
  className,
  overlap,
}: HeroBandProps) => {
  const t = TONES[tone];
  return (
    <section className={cn('relative overflow-hidden', t.section, className)}>
      <RadarRings
        tone={t.rings}
        sizes={[240, 400, 560]}
        strength={0.6}
        className="top-[100px] right-[calc(6%+280px)] max-md:right-0"
      />
      <div
        className={cn(
          'relative mx-auto flex max-w-[1280px] flex-col gap-4 px-4 pt-11 sm:px-8',
          overlap ? 'pb-28' : 'pb-10',
        )}
      >
        {eyebrow ? <Eyebrow className={t.eyebrow}>{eyebrow}</Eyebrow> : null}
        <h1 className="display-caps text-[clamp(40px,5vw,68px)]">{title}</h1>
        {description ? (
          <p className="max-w-[640px] text-[19px] opacity-90">{description}</p>
        ) : null}
        {children}
      </div>
    </section>
  );
};
