import type { ReactNode } from 'react';

import { AsyncState } from './AsyncState';
import { Skeleton } from './Skeleton';

interface RemoteStateProps {
  loading: boolean;
  error?: string | null;
  retry?: () => void;
  retryDisabled?: boolean;
  loadingLabel?: string;
  loadingContent?: ReactNode;
  errorTitle?: string;
  empty?: boolean;
  emptyTitle?: string;
  emptyDescription?: string;
  emptyContent?: ReactNode;
  className?: string;
  children?: ReactNode;
}

/** Нейтральный скелетон «три строки текста»: подходит в карточку, диалог и список. */
export const RemoteStateSkeleton = () => {
  return (
    <div className="flex flex-col gap-3">
      <Skeleton className="h-5 w-40 max-w-full" />
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-2/3" />
    </div>
  );
};

/** Показывает загрузку, ошибку, пустой список или содержимое. Фоновое обновление обрабатывает вызывающий компонент. */
export const RemoteState = ({
  loading,
  error,
  retry,
  retryDisabled,
  loadingLabel = 'Загрузка данных',
  loadingContent,
  errorTitle = 'Не удалось загрузить данные',
  empty = false,
  emptyTitle = 'Данных пока нет',
  emptyDescription,
  emptyContent,
  className,
  children,
}: RemoteStateProps) => {
  if (loading) {
    return (
      <div
        role="status"
        aria-label={loadingLabel}
        aria-busy="true"
        className={className}
      >
        <span className="sr-only">{loadingLabel}</span>
        <div
          aria-hidden="true"
          className="motion-reduce:[&_[data-slot=skeleton]]:animate-none"
        >
          {loadingContent ?? <RemoteStateSkeleton />}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <AsyncState
        error
        title={errorTitle}
        description={error}
        onRetry={retry}
        retryDisabled={retryDisabled}
        className={className}
      />
    );
  }

  if (empty) {
    return (
      emptyContent ?? (
        <AsyncState
          title={emptyTitle}
          description={emptyDescription}
          className={className}
        />
      )
    );
  }

  return children ?? null;
};
