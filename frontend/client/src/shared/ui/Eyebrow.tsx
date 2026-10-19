import type { ComponentProps } from 'react';
import { cn } from 'cn';

/** Мета-строка: JetBrains Mono, только капс. Дата, зона, раздел. */
export const Eyebrow = ({ className, ...props }: ComponentProps<'div'>) => {
  return (
    <div
      data-slot="eyebrow"
      className={cn(
        'font-mono text-sm font-medium tracking-[0.04em] uppercase',
        className,
      )}
      {...props}
    />
  );
};
