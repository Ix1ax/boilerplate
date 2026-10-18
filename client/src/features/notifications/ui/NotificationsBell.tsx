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

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={
              unread ? `Уведомления, непрочитанных: ${unread}` : 'Уведомления'
            }
            className="relative"
          />
        }
      >
        <BellIcon />
        {unread > 0 && (
          <span className="bg-destructive absolute -top-1 -right-1 min-w-4 rounded-full px-1 text-[10px] leading-4 text-white">
            {unread > 9 ? '9+' : unread}
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
