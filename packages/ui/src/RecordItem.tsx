import type { ComponentProps, HTMLAttributes, ReactNode } from 'react';
import { PencilIcon, Trash2Icon } from 'lucide-react';

import { cn } from './utils';

import { Button } from './Button';

interface RecordItemProps extends HTMLAttributes<HTMLElement> {
  as?: 'li' | 'div' | 'section';
  children: ReactNode;
}

type RecordItemRootProps = RecordItemProps;

const RecordItemRoot = ({
  as: Component = 'li',
  className,
  ...props
}: RecordItemRootProps) => {
  return (
    <Component
      className={cn(
        'flex min-w-0 flex-col gap-3 rounded-xl border border-border/60 bg-muted/50 p-4 sm:p-5',
        className,
      )}
      {...props}
    />
  );
};

interface RecordItemHeaderProps extends ComponentProps<'div'> {
  children: ReactNode;
}

const RecordItemHeader = ({
  className,
  children,
  ...props
}: RecordItemHeaderProps) => {
  return (
    <div
      className={cn(
        'flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start sm:justify-between',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
};

interface RecordItemContentProps extends ComponentProps<'div'> {
  children: ReactNode;
}

const RecordItemContent = ({
  className,
  children,
  ...props
}: RecordItemContentProps) => {
  return (
    <div
      className={cn('grid min-w-0 gap-1 [overflow-wrap:anywhere]', className)}
      {...props}
    >
      {children}
    </div>
  );
};

interface RecordItemHeadingProps extends ComponentProps<'div'> {
  children: ReactNode;
}

const RecordItemHeading = ({
  className,
  children,
  ...props
}: RecordItemHeadingProps) => {
  return (
    <div
      className={cn('flex min-w-0 flex-wrap items-center gap-2', className)}
      {...props}
    >
      {children}
    </div>
  );
};

interface RecordItemTitleProps extends ComponentProps<'h3'> {
  children: ReactNode;
}

const RecordItemTitle = ({
  className,
  children,
  ...props
}: RecordItemTitleProps) => {
  return (
    <h3 className={cn('text-base font-semibold', className)} {...props}>
      {children}
    </h3>
  );
};

interface RecordItemSubtitleProps extends ComponentProps<'p'> {
  children: ReactNode;
}

const RecordItemSubtitle = ({
  className,
  children,
  ...props
}: RecordItemSubtitleProps) => {
  if (!children) return null;
  return (
    <p className={cn('text-sm', className)} {...props}>
      {children}
    </p>
  );
};

interface RecordItemMetaProps extends ComponentProps<'p'> {
  children: ReactNode;
}

const RecordItemMeta = ({
  className,
  children,
  ...props
}: RecordItemMetaProps) => {
  if (!children) return null;
  return (
    <p className={cn('text-sm text-muted-foreground', className)} {...props}>
      {children}
    </p>
  );
};

interface RecordItemActionsContainerProps extends ComponentProps<'div'> {
  children: ReactNode;
}

const RecordItemActionsContainer = ({
  className,
  children,
  ...props
}: RecordItemActionsContainerProps) => {
  if (!children) return null;
  return (
    <div
      className={cn('flex shrink-0 flex-wrap items-center gap-2', className)}
      {...props}
    >
      {children}
    </div>
  );
};

interface RecordItemDescriptionProps extends ComponentProps<'div'> {
  children: ReactNode;
}

const RecordItemDescription = ({
  className,
  children,
  ...props
}: RecordItemDescriptionProps) => {
  if (!children) return null;
  return (
    <div
      className={cn(
        'border-t border-border/60 pt-3 text-sm leading-relaxed whitespace-pre-wrap [overflow-wrap:anywhere]',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
};

export const RecordItem = Object.assign(RecordItemRoot, {
  Header: RecordItemHeader,
  Content: RecordItemContent,
  Heading: RecordItemHeading,
  Title: RecordItemTitle,
  Subtitle: RecordItemSubtitle,
  Meta: RecordItemMeta,
  Actions: RecordItemActionsContainer,
  Description: RecordItemDescription,
});

interface RecordItemActionsProps {
  /** Доступное имя записи для подписей кнопок: «запись: Название». */
  label: string;
  disabled?: boolean;
  onEdit: () => void;
  onDelete: () => void;
}

/** Стандартные действия записи: «Изменить» и удаление иконкой. */
export const RecordItemActions = ({
  label,
  disabled,
  onEdit,
  onDelete,
}: RecordItemActionsProps) => {
  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={disabled}
        onClick={onEdit}
        aria-label={`Редактировать ${label}`}
      >
        <PencilIcon aria-hidden="true" />
        Изменить
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        className="text-muted-foreground hover:text-destructive"
        disabled={disabled}
        onClick={onDelete}
        aria-label={`Удалить ${label}`}
      >
        <Trash2Icon aria-hidden="true" />
      </Button>
    </>
  );
};
