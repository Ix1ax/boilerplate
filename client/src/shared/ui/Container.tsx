import type { ComponentProps } from 'react';
import { cn } from 'cn';

/** Центрированный контейнер с max-width и горизонтальными отступами. */
export const Container = ({ className, ...props }: ComponentProps<'div'>) => {
  return (
    <div
      data-slot="container"
      className={cn('mx-auto w-full max-w-[1280px] px-4 sm:px-8', className)}
      {...props}
    />
  );
};
