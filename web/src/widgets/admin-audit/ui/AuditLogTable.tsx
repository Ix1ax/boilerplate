import { useSearchParams } from 'react-router';
import { ScrollTextIcon } from 'lucide-react';

import { useAuditLog } from '@/entities/audit';

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

const METHODS = ['POST', 'PATCH', 'DELETE'] as const;

const formatDate = (iso: string) =>
  new Date(iso).toLocaleString('ru-RU', {
    dateStyle: 'short',
    timeStyle: 'medium',
  });

export const AuditLogTable = () => {
  const pagination = useOffsetPagination(20, 'audit_offset');
  const [searchParams, setSearchParams] = useSearchParams();

  const q = searchParams.get('audit_q') ?? '';
  const method = searchParams.get('audit_method') ?? '';
  const requestId = searchParams.get('audit_request_id') ?? '';

  const update = (key: string, value: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (value) next.set(key, value);
      else next.delete(key);
      next.delete('audit_offset');
      return next;
    });
  };

  const { data, isPending, isError, refetch } = useAuditLog({
    offset: pagination.offset,
    limit: pagination.pageSize,
    q: q || undefined,
    method: method || undefined,
    requestId: requestId || undefined,
  });

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={q}
          onChange={(event) => update('audit_q', event.target.value)}
          placeholder="Поиск по пути или email…"
          aria-label="Поиск в журнале"
          className="max-w-xs"
        />
        <Select
          value={method || 'all'}
          onValueChange={(value) =>
            update('audit_method', value === 'all' ? '' : (value ?? ''))
          }
        >
          <SelectTrigger aria-label="Фильтр по методу">
            <SelectValue placeholder="Все методы" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Все методы</SelectItem>
            {METHODS.map((value) => (
              <SelectItem key={value} value={value}>
                {value}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Input
          value={requestId}
          onChange={(event) => update('audit_request_id', event.target.value)}
          placeholder="request_id"
          aria-label="Фильтр по request_id"
          className="max-w-48"
        />
      </div>

      {isPending && <Skeleton className="h-40 w-full" />}

      {!isPending && isError && (
        <div role="alert">
          Не удалось загрузить журнал.{' '}
          <Button onClick={() => void refetch()}>Повторить</Button>
        </div>
      )}

      {!isPending && !isError && data.items.length === 0 && (
        <EmptyState
          icon={ScrollTextIcon}
          title="Записей не найдено"
          description="Измените условия фильтра."
        />
      )}

      {!isPending && !isError && data.items.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Время</TableHead>
              <TableHead>Пользователь</TableHead>
              <TableHead>Метод</TableHead>
              <TableHead>Путь</TableHead>
              <TableHead>Статус</TableHead>
              <TableHead>request_id</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.items.map((entry, index) => (
              <TableRow key={index}>
                <TableCell>{formatDate(entry.occurredAt)}</TableCell>
                <TableCell
                  className="max-w-40 truncate"
                  title={entry.actorEmail}
                >
                  {entry.actorEmail}
                </TableCell>
                <TableCell>{entry.method}</TableCell>
                <TableCell className="max-w-60 truncate" title={entry.path}>
                  {entry.path}
                </TableCell>
                <TableCell>
                  <Badge
                    variant={entry.status >= 400 ? 'destructive' : 'secondary'}
                  >
                    {entry.status}
                  </Badge>
                </TableCell>
                <TableCell
                  className="max-w-32 truncate font-mono text-xs"
                  title={entry.requestId}
                >
                  {entry.requestId}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {!isPending && !isError && data && (
        <Pagination
          label="Страницы журнала"
          total={data.total}
          hasPrev={pagination.hasPrev}
          hasNext={pagination.hasNext(data.total)}
          onPrev={pagination.toPrev}
          onNext={pagination.toNext}
        />
      )}
    </div>
  );
};
