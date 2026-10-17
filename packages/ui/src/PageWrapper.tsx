import type { ComponentProps } from 'react';

import { cn } from './utils';

const PAGE_WRAPPER_VARIANTS = {
  page: 'container-page page-rhythm flex flex-1 flex-col gap-8',
  landing: '',
  auth: 'w-full rounded-2xl border border-border bg-card p-6 shadow-sm sm:p-8',
  login:
    'container-page flex flex-1 items-center justify-center py-12 md:py-24',
  workspace: 'container-page pt-6 pb-8 md:pt-10 md:pb-10',
} as const;

interface PageWrapperProps extends ComponentProps<'main'> {
  variant?: keyof typeof PAGE_WRAPPER_VARIANTS;
}

export const PageWrapper = ({
  variant = 'page',
  className,
  ...props
}: PageWrapperProps) => {
  return (
    <main
      className={cn(PAGE_WRAPPER_VARIANTS[variant], className)}
      {...props}
    />
  );
};
