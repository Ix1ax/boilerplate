import type { ComponentProps, ReactNode } from 'react';

import { cn } from './utils';

interface PageHeaderProps extends ComponentProps<'header'> {
  children: ReactNode;
}

interface PageHeaderRowProps extends ComponentProps<'div'> {
  children: ReactNode;
}

interface PageHeaderContentProps extends ComponentProps<'div'> {
  children: ReactNode;
}

interface PageHeaderTitleProps extends ComponentProps<'h1'> {
  children: ReactNode;
}

interface PageHeaderDescriptionProps extends ComponentProps<'div'> {
  children: ReactNode;
}

interface PageHeaderActionsProps extends ComponentProps<'div'> {
  children: ReactNode;
}

type PageHeaderRootProps = PageHeaderProps;

const PageHeaderRoot = ({ className, ...props }: PageHeaderRootProps) => {
  return (
    <header
      className={cn(
        'flex min-w-0 flex-col gap-4 rounded-2xl border border-border/60 bg-card p-5 sm:p-6',
        className,
      )}
      {...props}
    />
  );
};

const PageHeaderRow = ({ className, ...props }: PageHeaderRowProps) => {
  return (
    <div
      className={cn(
        'flex min-w-0 flex-col gap-4 lg:flex-row lg:items-start lg:justify-between lg:gap-8',
        className,
      )}
      {...props}
    />
  );
};

const PageHeaderContent = ({ className, ...props }: PageHeaderContentProps) => {
  return (
    <div className={cn('grid min-w-0 flex-1 gap-2', className)} {...props} />
  );
};

const PageHeaderTitle = ({ className, ...props }: PageHeaderTitleProps) => {
  return (
    <h1
      className={cn(
        'text-2xl font-semibold tracking-tight text-balance [overflow-wrap:anywhere] sm:text-3xl',
        className,
      )}
      {...props}
    />
  );
};

const PageHeaderDescription = ({
  className,
  children,
  ...props
}: PageHeaderDescriptionProps) => {
  if (!children) return null;
  return (
    <div
      className={cn(
        'max-w-3xl text-base leading-relaxed text-muted-foreground',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
};

const PageHeaderActions = ({
  className,
  children,
  ...props
}: PageHeaderActionsProps) => {
  if (!children) return null;
  return (
    <div
      className={cn(
        'flex min-w-0 shrink-0 flex-wrap items-start gap-3 lg:max-w-sm lg:justify-end',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
};

export const PageHeader = Object.assign(PageHeaderRoot, {
  Row: PageHeaderRow,
  Content: PageHeaderContent,
  Title: PageHeaderTitle,
  Description: PageHeaderDescription,
  Actions: PageHeaderActions,
});
