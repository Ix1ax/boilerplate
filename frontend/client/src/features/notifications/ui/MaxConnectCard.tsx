import { Button, Notice } from '@/shared/ui';

import { useMax } from '../model';

/** Карточка для страницы настроек: подключить или отключить напоминания в мессенджере MAX. */
export const MaxConnectCard = () => {
  const max = useMax();

  if (!max.status?.enabled) return null;

  return (
    <div className="flex flex-wrap items-center justify-between gap-4 rounded-2xl bg-card p-5">
      <div>
        <b className="text-lg">Мессенджер MAX</b>
        <p className="mt-1 text-sm text-muted-foreground">
          Напоминания о встречах и ответы организатора — прямо в MAX.
        </p>
      </div>
      {max.status.linked ? (
        <Button
          variant="outline"
          disabled={max.unlink.isPending}
          onClick={() => max.unlink.mutate()}
        >
          Отключить
        </Button>
      ) : (
        <Button disabled={max.link.isPending} onClick={() => max.link.mutate()}>
          Подключить MAX
        </Button>
      )}
      {max.unlink.isError && (
        <Notice role="alert" tone="danger" className="w-full">
          Не удалось отключить MAX. Попробуйте ещё раз.
        </Notice>
      )}
    </div>
  );
};
