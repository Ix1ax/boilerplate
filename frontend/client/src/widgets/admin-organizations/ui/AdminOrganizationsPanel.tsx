import { useState } from 'react';
import { Link, useSearchParams } from 'react-router';
import { Building2Icon } from 'lucide-react';

import { ReasonDialog, useApproveOrganization, useRejectOrganization } from '@/features/admin-moderation';

import { useAdminOrganizations } from '@/entities/organization';

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

const STATUS_LABEL: Record<string, string> = {
  pending: 'На рассмотрении',
  approved: 'Допущена',
  rejected: 'Отклонена',
};

const STATUS_BADGE: Record<string, 'warning' | 'success' | 'destructive'> = {
  pending: 'warning',
  approved: 'success',
  rejected: 'destructive',
};

const formatDate = (iso: string) =>
  new Date(iso).toLocaleDateString('ru-RU', { dateStyle: 'medium' });

export const AdminOrganizationsPanel = () => {
  const pagination = useOffsetPagination(20, 'org_offset');
  const [searchParams, setSearchParams] = useSearchParams();
  const q = searchParams.get('org_q') ?? '';
  const status = searchParams.get('org_status') ?? 'pending';
  const [rejectId, setRejectId] = useState<string | null>(null);

  const update = (key: string, value: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (value) next.set(key, value);
      else next.delete(key);
      next.delete('org_offset');
      return next;
    });
  };

  const { data, isPending, isError, refetch } = useAdminOrganizations({
    offset: pagination.offset,
    limit: pagination.pageSize,
    q: q || undefined,
    status: status === 'all' ? undefined : status,
  });

  const approve = useApproveOrganization();
  const reject = useRejectOrganization();

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={q}
          onChange={(event) => update('org_q', event.target.value)}
          placeholder="Поиск по названию, описанию, городу…"
          aria-label="Поиск организаций"
          className="max-w-xs"
        />
        <Select
          value={status}
          onValueChange={(value) => update('org_status', value ?? 'pending')}
        >
          <SelectTrigger aria-label="Фильтр по статусу">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="pending">На рассмотрении</SelectItem>
            <SelectItem value="approved">Допущены</SelectItem>
            <SelectItem value="rejected">Отклонены</SelectItem>
            <SelectItem value="all">Все статусы</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isPending && <Skeleton className="h-40 w-full" />}

      {!isPending && isError && (
        <div role="alert">
          Не удалось загрузить организации.{' '}
          <Button onClick={() => void refetch()}>Повторить</Button>
        </div>
      )}

      {!isPending && !isError && data.items.length === 0 && (
        <EmptyState
          icon={Building2Icon}
          title="Организаций не найдено"
          description="Измените условия фильтра."
        />
      )}

      {!isPending && !isError && data.items.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Клуб</TableHead>
              <TableHead>Город</TableHead>
              <TableHead>Статус</TableHead>
              <TableHead>Представитель</TableHead>
              <TableHead>Подана</TableHead>
              <TableHead>Действия</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.items.map((org) => (
              <TableRow key={org.id}>
                <TableCell>
                  <Link
                    to={DYNAMIC_ROUTES.ORGANIZATION(org.id)}
                    className="font-bold underline-offset-3 hover:underline"
                  >
                    {org.name}
                  </Link>
                </TableCell>
                <TableCell>{org.city}</TableCell>
                <TableCell>
                  <Badge variant={STATUS_BADGE[org.status]}>
                    {STATUS_LABEL[org.status]}
                  </Badge>
                </TableCell>
                <TableCell className="max-w-48 truncate" title={org.owner?.name}>
                  {org.owner?.name ?? '—'}
                </TableCell>
                <TableCell>{formatDate(org.created_at)}</TableCell>
                <TableCell>
                  {org.status === 'pending' && (
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        disabled={approve.isPending}
                        onClick={() => approve.mutate(org.id)}
                      >
                        Допустить
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setRejectId(org.id)}
                      >
                        Отклонить
                      </Button>
                    </div>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {!isPending && !isError && data && (
        <Pagination
          label="Страницы заявок"
          total={data.total}
          hasPrev={pagination.hasPrev}
          hasNext={pagination.hasNext(data.total)}
          onPrev={pagination.toPrev}
          onNext={pagination.toNext}
        />
      )}

      <ReasonDialog
        open={rejectId !== null}
        onOpenChange={(open) => !open && setRejectId(null)}
        title="Отклонить заявку"
        description="Причина обязательна — представитель увидит её и сможет исправить профиль."
        submitLabel="Отклонить"
        destructive
        pending={reject.isPending}
        error={reject.error}
        onSubmit={(reason) => {
          if (!rejectId) return;
          reject.mutate(
            { id: rejectId, reason },
            { onSuccess: () => setRejectId(null) },
          );
        }}
      />
    </div>
  );
};
