import { Link } from 'react-router';
import { BellIcon } from 'lucide-react';

import {
  Button,
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/shared/ui';

import { useMarkRead, useMax, useNotifications } from '../model';

export const NotificationsBell = () => {
  const { data } = useNotifications();
  const markRead = useMarkRead();
  const max = useMax();
  const unread = data?.unread ?? 0;
  // Одна строка для значка и имени кнопки: доступное имя должно содержать видимый текст (WCAG 2.5.3).
  const badge = unread > 9 ? '9+' : String(unread);

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            size="icon"
            aria-label={
              unread ? `Уведомления, непрочитанных: ${badge}` : 'Уведомления'
            }
            className="relative"
          />
        }
      >
        <BellIcon />
        {unread > 0 && (
          <span className="absolute top-0.5 right-0.5 min-w-5 rounded-full bg-gold px-1 text-xs leading-5 font-bold text-ink">
            {badge}
          </span>
        )}
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80">
        <PopoverHeader className="flex-row items-center justify-between">
          <PopoverTitle>Уведомления</PopoverTitle>
          {unread > 0 && (
            <Button
              variant="link"
              size="sm"
              onClick={() => markRead.mutate(undefined)}
            >
              Прочитать все
            </Button>
          )}
        </PopoverHeader>
        {!data?.items.length ? (
          <p className="text-muted-foreground py-4 text-center text-sm">
            Пока ничего нет
          </p>
        ) : (
          <ul className="max-h-80 space-y-1 overflow-y-auto">
            {data.items.map((item) => (
              <li key={item.id}>
                <Link
                  to={item.link || '#'}
                  onClick={() => !item.read_at && markRead.mutate(item.id)}
                  className="hover:bg-muted block rounded-md p-2 text-sm"
                >
                  <span className={item.read_at ? '' : 'font-semibold'}>
                    {item.title}
                  </span>
                  {item.body && (
                    <span className="text-muted-foreground block text-xs">
                      {item.body}
                    </span>
                  )}
                </Link>
              </li>
            ))}
          </ul>
        )}
        {max.status?.enabled && (
          <div className="flex items-center justify-between border-t pt-2 text-sm">
            {max.status.linked ? (
              <>
                <span className="text-muted-foreground">MAX подключён</span>
                <Button
                  variant="link"
                  size="sm"
                  disabled={max.unlink.isPending}
                  onClick={() => max.unlink.mutate()}
                >
                  Отключить
                </Button>
              </>
            ) : (
              <Button
                variant="outline"
                size="sm"
                className="w-full"
                disabled={max.link.isPending}
                onClick={() => max.link.mutate()}
              >
                Получать уведомления в MAX
              </Button>
            )}
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
};
