import type { ComponentProps, ReactNode } from 'react';
import { cn } from 'cn';

interface PageHeaderProps extends Omit<ComponentProps<'div'>, 'title'> {
  title: ReactNode;
  description?: ReactNode;
  /** Контролы справа (кнопки действий). */
  actions?: ReactNode;
}

/** Шапка страницы: заголовок + описание слева, действия справа. */
export const PageHeader = ({
  title,
  description,
  actions,
  className,
  ...props
}: PageHeaderProps) => {
  return (
    <div
      data-slot="page-header"
      className={cn(
        'flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between',
        className,
      )}
      {...props}
    >
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        {description ? (
          <p className="text-sm text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {actions ? (
        <div className="flex items-center gap-2">{actions}</div>
      ) : null}
    </div>
  );
};
