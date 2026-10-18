import { cn } from 'cn';
import { Loader2Icon } from 'lucide-react';

interface LoaderScreenProps {
  label?: string;
  className?: string;
}

/** Полноэкранный лоадер с блюром фона. Для роут-фолбэков и проверки сессии. */
export const LoaderScreen = ({
  label = 'Загрузка…',
  className,
}: LoaderScreenProps) => {
  return (
    <div
      data-slot="loader-screen"
      role="status"
      aria-live="polite"
      aria-busy="true"
      className={cn(
        'fixed inset-0 z-50 flex flex-col items-center justify-center gap-3 bg-background/60 backdrop-blur-sm',
        className,
      )}
    >
      <Loader2Icon
        aria-hidden
        className="size-8 animate-spin text-muted-foreground motion-reduce:animate-none"
      />
      {label ? (
        <span className="text-sm text-muted-foreground">{label}</span>
      ) : null}
    </div>
  );
};
