import { HistoryIcon } from 'lucide-react';

import { STATUS_META, useTaskHistory } from '@/entities/task';

import {
  Button,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  EmptyState,
  Skeleton,
} from '@/shared/ui';

interface TaskHistoryDialogProps {
  taskId: string;
  taskTitle: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const formatDate = (iso: string) =>
  new Date(iso).toLocaleString('ru-RU', {
    dateStyle: 'short',
    timeStyle: 'short',
  });

export const TaskHistoryDialog = ({
  taskId,
  taskTitle,
  open,
  onOpenChange,
}: TaskHistoryDialogProps) => {
  const { data, isPending } = useTaskHistory(taskId, open);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>История статусов</DialogTitle>
        </DialogHeader>
        <p className="truncate text-sm text-muted-foreground" title={taskTitle}>
          {taskTitle}
        </p>
        {isPending && <Skeleton className="h-20 w-full" />}
        {!isPending && data && data.length > 0 && (
          <ol className="space-y-3">
            {data.map((transition, index) => (
              <li key={index} className="border-l-2 border-border pl-3 text-sm">
                <p>
                  <span className="text-muted-foreground">
                    {STATUS_META[transition.from]?.label ?? transition.from}
                  </span>{' '}
                  →{' '}
                  <span className="font-medium">
                    {STATUS_META[transition.to]?.label ?? transition.to}
                  </span>
                </p>
                <p className="text-xs text-muted-foreground">
                  {formatDate(transition.createdAt)} · {transition.actor}
                </p>
                {transition.comment && (
                  <p className="mt-1">{transition.comment}</p>
                )}
              </li>
            ))}
          </ol>
        )}
        {!isPending && (!data || data.length === 0) && (
          <EmptyState icon={HistoryIcon} title="Переходов пока не было" />
        )}
        <Button variant="outline" onClick={() => onOpenChange(false)}>
          Закрыть
        </Button>
      </DialogContent>
    </Dialog>
  );
};
