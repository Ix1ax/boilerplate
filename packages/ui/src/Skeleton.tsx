import type { ComponentProps } from 'react';

import { cn } from './utils';

type SkeletonProps = ComponentProps<'div'>;

const Skeleton = ({ className, ...props }: SkeletonProps) => {
  return (
    <div
      data-slot="skeleton"
      className={cn('animate-pulse rounded-md bg-foreground/8', className)}
      {...props}
    />
  );
};

export { Skeleton };
