import { useState } from 'react';
import { Link, useSearchParams } from 'react-router';
import { FlagIcon } from 'lucide-react';

import { ReasonDialog, useDismissReport } from '@/features/admin-moderation';

import { useAdminReports } from '@/entities/report';

import { DYNAMIC_ROUTES } from '@/shared/constants';
import { useOffsetPagination } from '@/shared/hooks';
import {
  Badge,
  Button,
  EmptyState,
  Pagination,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
} from '@/shared/ui';

const REASON_LABEL: Record<string, string> = {
  spam: 'Спам',
  offensive: 'Оскорбление',
  fraud: 'Обман',
  wrong_info: 'Неверная информация',
  dangerous: 'Опасно',
  other: 'Другое',
};

const STATUS_LABEL: Record<string, string> = {
  open: 'Ждёт решения',
  resolved: 'Меры приняты',
  dismissed: 'Отклонена',
};

const STATUS_BADGE: Record<string, 'warning' | 'success' | 'neutral'> = {
  open: 'warning',
  resolved: 'success',
  dismissed: 'neutral',
};

const formatDate = (iso: string) =>
  new Date(iso).toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });

export const AdminReportsPanel = () => {
  const pagination = useOffsetPagination(20, 'rep_offset');
  const [searchParams, setSearchParams] = useSearchParams();
  const status = searchParams.get('rep_status') ?? 'open';
  const [dismissId, setDismissId] = useState<string | null>(null);

  const update = (key: string, value: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (value) next.set(key, value);
      else next.delete(key);
      next.delete('rep_offset');
      return next;
    });
  };

  const { data, isPending, isError, refetch } = useAdminReports({
    offset: pagination.offset,
    limit: pagination.pageSize,
    status: status === 'all' ? undefined : (status as 'open' | 'resolved' | 'dismissed'),
  });

  const dismiss = useDismissReport();

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <Select
          value={status}
          onValueChange={(value) => update('rep_status', value ?? 'open')}
        >
          <SelectTrigger aria-label="Фильтр по статусу">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="open">Ждут решения</SelectItem>
            <SelectItem value="resolved">Меры приняты</SelectItem>
            <SelectItem value="dismissed">Отклонены</SelectItem>
            <SelectItem value="all">Все</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isPending && <Skeleton className="h-40 w-full" />}

      {!isPending && isError && (
        <div role="alert">
          Не удалось загрузить жалобы.{' '}
          <Button onClick={() => void refetch()}>Повторить</Button>
        </div>
      )}

      {!isPending && !isError && data.items.length === 0 && (
        <EmptyState icon={FlagIcon} title="Жалоб не найдено" />
      )}

      {!isPending && !isError && data.items.length > 0 && (
        <ul className="space-y-3">
          {data.items.map((report) => (
            <li
              key={report.id}
              className="flex flex-col gap-2 rounded-2xl border border-border bg-card p-4"
            >
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant={STATUS_BADGE[report.status]}>
                    {STATUS_LABEL[report.status]}
                  </Badge>
                  <Badge variant="outline">
                    {REASON_LABEL[report.reason] ?? report.reason}
                  </Badge>
                  {report.open_on_target > 1 && (
                    <Badge variant="destructive">
                      Жалоб на эту цель: {report.open_on_target}
                    </Badge>
                  )}
                </div>
                <span className="text-xs text-muted-foreground">
                  {formatDate(report.created_at)}
                </span>
              </div>
              <p className="font-bold">{report.target.title}</p>
              <p className="text-sm text-muted-foreground line-clamp-2">
                {report.target.text}
              </p>
              {report.details && <p className="text-sm">«{report.details}»</p>}
              <p className="text-sm text-muted-foreground">
                Автор жалобы: {report.reporter.name}
              </p>
              <div className="flex flex-wrap gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  render={<Link to={DYNAMIC_ROUTES.EVENT(report.event_id)} />}
                >
                  К мероприятию
                </Button>
                {report.status === 'open' && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => setDismissId(report.id)}
                  >
                    Отклонить жалобу
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      {!isPending && !isError && data && (
        <Pagination
          label="Страницы жалоб"
          total={data.total}
          hasPrev={pagination.hasPrev}
          hasNext={pagination.hasNext(data.total)}
          onPrev={pagination.toPrev}
          onNext={pagination.toNext}
        />
      )}

      <ReasonDialog
        open={dismissId !== null}
        onOpenChange={(open) => !open && setDismissId(null)}
        title="Отклонить жалобу"
        description="Закроет все открытые жалобы на эту цель; пояснение увидят их авторы."
        submitLabel="Отклонить"
        pending={dismiss.isPending}
        error={dismiss.error}
        onSubmit={(resolution) => {
          if (!dismissId) return;
          dismiss.mutate(
            { id: dismissId, resolution },
            { onSuccess: () => setDismissId(null) },
          );
        }}
      />
    </div>
  );
};
