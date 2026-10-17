import { cn } from 'cn';
import { AlertTriangleIcon } from 'lucide-react';

import { Button } from './Button';

interface ErrorScreenProps {
  title?: string;
  description?: string;
  onRetry?: () => void;
  retryLabel?: string;
  className?: string;
}

/** Полноэкранное состояние ошибки с повтором. Для роут-фолбэков и проверки сессии. */
export const ErrorScreen = ({
  title = 'Что-то пошло не так',
  description = 'Не удалось загрузить данные. Попробуйте ещё раз.',
  onRetry,
  retryLabel = 'Повторить',
  className,
}: ErrorScreenProps) => {
  return (
    <div
      data-slot="error-screen"
      role="alert"
      className={cn(
        'fixed inset-0 z-50 flex flex-col items-center justify-center gap-4 bg-background/60 px-6 text-center backdrop-blur-sm',
        className,
      )}
    >
      <AlertTriangleIcon aria-hidden className="size-8 text-destructive" />
      <div className="space-y-1">
        <p className="text-base font-medium">{title}</p>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
      {onRetry ? (
        <Button variant="outline" onClick={onRetry}>
          {retryLabel}
        </Button>
      ) : null}
    </div>
  );
};
