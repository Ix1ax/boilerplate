import type { ComponentProps } from 'react';

import { cn } from './utils';
type MetaLabelProps = ComponentProps<'span'>;

export const MetaLabel = ({ className, ...props }: MetaLabelProps) => {
  return (
    <span
      className={cn(
        'font-mono text-[10px] uppercase tracking-wider text-caption',
        className,
      )}
      {...props}
    />
  );
};
