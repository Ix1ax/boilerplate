import type { ComponentProps } from 'react';
import { cn } from 'cn';

/** Вертикальный блок-секция с отступами между детьми. */
export const Section = ({ className, ...props }: ComponentProps<'section'>) => {
  return (
    <section
      data-slot="section"
      className={cn('flex flex-col gap-4 py-6', className)}
      {...props}
    />
  );
};
