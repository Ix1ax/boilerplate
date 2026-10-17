import type { ComponentProps } from 'react';

import { cn } from './utils';

type RadioProps = Omit<ComponentProps<'input'>, 'type'>;

export const Radio = ({ className, ...props }: RadioProps) => {
  return (
    <input
      {...props}
      type="radio"
      data-slot="radio"
      className={cn(
        'size-4 shrink-0 cursor-pointer accent-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
    />
  );
};
