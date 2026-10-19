import { cn } from 'cn';

interface BrandMarkProps {
  /** Фон под знаком: обводка золотого круга повторяет его цвет. */
  tone?: 'light' | 'dark';
  /** Подпись под словом «вместе» — только на лендинге. */
  tagline?: string;
  className?: string;
}

/** Знак «вместе»: синий и золотой круг сходятся — двое нашли друг друга. */
export const BrandMark = ({
  tone = 'light',
  tagline,
  className,
}: BrandMarkProps) => {
  return (
    <span className={cn('flex items-center gap-3', className)}>
      <span aria-hidden="true" className="relative block h-8 w-12 shrink-0">
        <span className="absolute top-0 left-0 size-8 rounded-full bg-blue" />
        <span
          className={cn(
            'absolute top-0 left-4 size-8 rounded-full bg-gold',
            tone === 'dark'
              ? 'shadow-[0_0_0_3px_var(--color-ink)]'
              : 'shadow-[0_0_0_3px_var(--color-card)]',
          )}
        />
      </span>
      <span className="flex flex-col gap-[3px]">
        <span className="font-display text-[21px] leading-none font-extrabold tracking-[-0.02em]">
          вместе
        </span>
        {tagline ? (
          <span className="text-[13px] leading-tight opacity-75">
            {tagline}
          </span>
        ) : null}
      </span>
    </span>
  );
};
