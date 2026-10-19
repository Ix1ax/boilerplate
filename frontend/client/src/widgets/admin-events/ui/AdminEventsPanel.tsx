import { useState } from 'react';
import { Link, useSearchParams } from 'react-router';
import { CalendarXIcon } from 'lucide-react';

import {
  ReasonDialog,
  useDeleteEvent,
  useRestoreEvent,
  useUnpublishEvent,
} from '@/features/admin-moderation';

import { useAdminEvents } from '@/entities/event';

import { DYNAMIC_ROUTES } from '@/shared/constants';
import { useOffsetPagination } from '@/shared/hooks';
import {
  Badge,
  Button,
  EmptyState,
  Input,
  Pagination,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/shared/ui';

import { AdminEventHistoryDialog } from './AdminEventHistoryDialog';

const STATUS_LABEL: Record<string, string> = {
  published: 'Опубликовано',
  unpublished: 'Снято',
  deleted: 'Удалено',
  cancelled: 'Отменено',
};

const STATUS_BADGE: Record<string, 'success' | 'warning' | 'destructive' | 'neutral'> = {
  published: 'success',
  unpublished: 'warning',
  deleted: 'destructive',
  cancelled: 'neutral',
};

const formatDate = (iso: string) =>
  new Date(iso).toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });

export const AdminEventsPanel = () => {
  const pagination = useOffsetPagination(20, 'ev_offset');
  const [searchParams, setSearchParams] = useSearchParams();
  const q = searchParams.get('ev_q') ?? '';
  const status = searchParams.get('ev_status') ?? 'all';
  const [unpublishId, setUnpublishId] = useState<string | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [historyId, setHistoryId] = useState<string | null>(null);

  const update = (key: string, value: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (value) next.set(key, value);
      else next.delete(key);
      next.delete('ev_offset');
      return next;
    });
  };

  const { data, isPending, isError, refetch } = useAdminEvents({
    offset: pagination.offset,
    limit: pagination.pageSize,
    q: q || undefined,
    status: status === 'all' ? undefined : status,
  });

  const unpublish = useUnpublishEvent();
  const remove = useDeleteEvent();
  const restore = useRestoreEvent();

  const historyEvent = data?.items.find((event) => event.id === historyId);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={q}
          onChange={(event) => update('ev_q', event.target.value)}
          placeholder="Поиск по названию…"
          aria-label="Поиск мероприятий"
          className="max-w-xs"
        />
        <Select
          value={status}
          onValueChange={(value) => update('ev_status', value === 'all' ? '' : (value ?? ''))}
        >
          <SelectTrigger aria-label="Фильтр по статусу">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Все статусы</SelectItem>
            <SelectItem value="published">Опубликованные</SelectItem>
            <SelectItem value="unpublished">Снятые</SelectItem>
            <SelectItem value="deleted">Удалённые</SelectItem>
            <SelectItem value="cancelled">Отменённые</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isPending && <Skeleton className="h-40 w-full" />}

      {!isPending && isError && (
        <div role="alert">
          Не удалось загрузить мероприятия.{' '}
          <Button onClick={() => void refetch()}>Повторить</Button>
        </div>
      )}

      {!isPending && !isError && data.items.length === 0 && (
        <EmptyState
          icon={CalendarXIcon}
          title="Мероприятий не найдено"
          description="Измените условия фильтра."
        />
      )}

      {!isPending && !isError && data.items.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Мероприятие</TableHead>
              <TableHead>Организатор</TableHead>
              <TableHead>Когда</TableHead>
              <TableHead>Статус</TableHead>
              <TableHead>Действия</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.items.map((event) => (
              <TableRow key={event.id}>
                <TableCell>
                  <Link
                    to={DYNAMIC_ROUTES.EVENT(event.id)}
                    className="font-bold underline-offset-3 hover:underline"
                  >
                    {event.title}
                  </Link>
                </TableCell>
                <TableCell className="max-w-48 truncate" title={event.organizer.name}>
                  {event.organizer.name}
                </TableCell>
                <TableCell>{formatDate(event.startsAt)}</TableCell>
                <TableCell>
                  <Badge variant={STATUS_BADGE[event.status]}>
                    {STATUS_LABEL[event.status]}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-2">
                    {event.status === 'published' && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setUnpublishId(event.id)}
                      >
                        Снять
                      </Button>
                    )}
                    {event.status !== 'deleted' && (
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => setDeleteId(event.id)}
                      >
                        Удалить
                      </Button>
                    )}
                    {event.status === 'unpublished' && (
                      <Button
                        size="sm"
                        disabled={restore.isPending}
                        onClick={() => restore.mutate(event.id)}
                      >
                        Вернуть
                      </Button>
                    )}
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => setHistoryId(event.id)}
                    >
                      История
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {!isPending && !isError && data && (
        <Pagination
          label="Страницы мероприятий"
          total={data.total}
          hasPrev={pagination.hasPrev}
          hasNext={pagination.hasNext(data.total)}
          onPrev={pagination.toPrev}
          onNext={pagination.toNext}
        />
      )}

      <ReasonDialog
        open={unpublishId !== null}
        onOpenChange={(open) => !open && setUnpublishId(null)}
        title="Снять мероприятие с публикации"
        description="Запись закроется, организатор и участники получат уведомление с причиной."
        submitLabel="Снять"
        pending={unpublish.isPending}
        error={unpublish.error}
        onSubmit={(reason) => {
          if (!unpublishId) return;
          unpublish.mutate(
            { id: unpublishId, reason },
            { onSuccess: () => setUnpublishId(null) },
          );
        }}
      />

      <ReasonDialog
        open={deleteId !== null}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Удалить мероприятие"
        description="Мягкое удаление с причиной; вернуть нельзя. Записи сохраняются для истории."
        submitLabel="Удалить"
        destructive
        pending={remove.isPending}
        error={remove.error}
        onSubmit={(reason) => {
          if (!deleteId) return;
          remove.mutate(
            { id: deleteId, reason },
            { onSuccess: () => setDeleteId(null) },
          );
        }}
      />

      {historyId && (
        <AdminEventHistoryDialog
          eventId={historyId}
          eventTitle={historyEvent?.title ?? ''}
          open={historyId !== null}
          onOpenChange={(open) => !open && setHistoryId(null)}
        />
      )}
    </div>
  );
};
