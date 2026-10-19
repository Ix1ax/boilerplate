import { HistoryIcon } from 'lucide-react';

import { useAdminEventHistory } from '@/entities/event';

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  EmptyState,
  Skeleton,
} from '@/shared/ui';

interface AdminEventHistoryDialogProps {
  eventId: string;
  eventTitle: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const STATUS_LABEL: Record<string, string> = {
  published: 'Опубликовано',
  unpublished: 'Снято',
  deleted: 'Удалено',
  cancelled: 'Отменено',
  gathering: 'Набирается состав',
  reached: 'Состав собран',
  failed: 'Не набрали состав',
};

const formatDate = (iso: string) =>
  new Date(iso).toLocaleString('ru-RU', {
    dateStyle: 'short',
    timeStyle: 'short',
  });

export const AdminEventHistoryDialog = ({
  eventId,
  eventTitle,
  open,
  onOpenChange,
}: AdminEventHistoryDialogProps) => {
  const { data, isPending } = useAdminEventHistory(eventId, open);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>История мероприятия</DialogTitle>
        </DialogHeader>
        <p className="truncate text-sm text-muted-foreground" title={eventTitle}>
          {eventTitle}
        </p>
        {isPending && <Skeleton className="h-20 w-full" />}
        {!isPending && data && data.length > 0 && (
          <ol className="space-y-3">
            {data.map((entry, index) => (
              <li key={index} className="border-l-2 border-border pl-3 text-sm">
                <p>
                  <span className="text-muted-foreground">
                    {STATUS_LABEL[entry.from] ?? entry.from}
                  </span>{' '}
                  →{' '}
                  <span className="font-medium">
                    {STATUS_LABEL[entry.to] ?? entry.to}
                  </span>
                </p>
                <p className="text-xs text-muted-foreground">
                  {formatDate(entry.created_at)} · {entry.actor}
                </p>
                {entry.comment && <p className="mt-1">{entry.comment}</p>}
              </li>
            ))}
          </ol>
        )}
        {!isPending && (!data || data.length === 0) && (
          <EmptyState icon={HistoryIcon} title="Переходов пока не было" />
        )}
      </DialogContent>
    </Dialog>
  );
};
