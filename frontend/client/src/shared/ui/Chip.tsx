import type { ComponentProps } from 'react';
import { cn } from 'cn';

interface ChipProps extends ComponentProps<'button'> {
  pressed: boolean;
}

/** Переключатель-фильтр: выбранный — чёрный, остальные — с серой рамкой. */
export const Chip = ({ pressed, className, ...props }: ChipProps) => {
  return (
    <button
      type="button"
      aria-pressed={pressed}
      className={cn(
        'inline-flex min-h-11 items-center gap-2 rounded-full px-[18px] text-base transition-colors',
        pressed
          ? 'bg-ink font-bold text-paper dark:bg-paper dark:text-ink'
          : 'border-[1.5px] border-line-strong bg-card font-semibold text-foreground hover:border-foreground dark:border-border',
        className,
      )}
      {...props}
    />
  );
};
