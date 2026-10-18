import { useState } from 'react';
import { useSearchParams } from 'react-router';
import {
  HistoryIcon,
  ListTodoIcon,
  SparklesIcon,
  Trash2Icon,
} from 'lucide-react';

import { useConfirm } from '@/features/confirm';

import { type TaskStatus, useTasks } from '@/entities/task';

import { useOffsetPagination } from '@/shared/hooks';
import {
  Badge,
  Button,
  EmptyState,
  Pagination,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/shared/ui';

import { useAnalyzeTask, useDeleteTask } from '../model';
import { TaskFilters } from './TaskFilters';
import { TaskHistoryDialog } from './TaskHistoryDialog';
import { TaskStatusBadge } from './TaskStatusBadge';
import { TaskStatusMenu } from './TaskStatusMenu';

export const TaskTable = ({
  onContinueDraft,
}: {
  onContinueDraft: (id: string) => void;
}) => {
  const pagination = useOffsetPagination();
  const [searchParams] = useSearchParams();
  const [historyTaskId, setHistoryTaskId] = useState<string | null>(null);
  const deleteTask = useDeleteTask();
  const analyze = useAnalyzeTask();
  const confirm = useConfirm();

  const q = searchParams.get('q') || undefined;
  const status = (searchParams.get('status') || undefined) as
    TaskStatus | undefined;
  const sort = searchParams.get('sort') || undefined;

  const { data, isPending, isError, refetch } = useTasks({
    offset: pagination.offset,
    limit: pagination.pageSize,
    q,
    status,
    sort,
  });

  const historyTask = data?.items.find((task) => task.id === historyTaskId);
  const isPaginationVisible = !isPending && !isError && data;

  const handleDelete = async (id: string, title: string) => {
    const ok = await confirm({
      title: 'Удалить задачу?',
      description: title,
      destructive: true,
    });
    if (ok) deleteTask.mutate(id);
  };

  return (
    <div className="space-y-4">
      <TaskFilters />

      {isPending && <Skeleton className="h-40 w-full" />}

      {!isPending && isError && (
        <div role="alert">
          Не удалось загрузить задачи.{' '}
          <Button onClick={() => void refetch()}>Повторить</Button>
        </div>
      )}

      {!isPending && !isError && data.items.length === 0 && (
        <EmptyState
          icon={ListTodoIcon}
          title="Задач не найдено"
          description="Измените условия поиска или создайте новую задачу."
        />
      )}

      {!isPending && !isError && data.items.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Название</TableHead>
              <TableHead>Статус</TableHead>
              <TableHead>Проверка ИИ</TableHead>
              <TableHead className="text-right">Действия</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.items.map((task) => (
              <TableRow key={task.id}>
                <TableCell className="max-w-xs truncate" title={task.title}>
                  {task.title}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <TaskStatusBadge status={task.status} />
                    <TaskStatusMenu taskId={task.id} status={task.status} />
                  </div>
                </TableCell>
                <TableCell>
                  {task.moderation ? (
                    <Badge
                      variant={
                        task.moderation.verdict === 'ok'
                          ? 'secondary'
                          : 'destructive'
                      }
                    >
                      {task.moderation.verdict === 'ok'
                        ? 'Замечаний нет'
                        : 'Нужна проверка'}
                    </Badge>
                  ) : (
                    <span className="text-xs text-muted-foreground">
                      не проверена
                    </span>
                  )}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    {task.status === 'draft' && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onContinueDraft(task.id)}
                      >
                        Продолжить черновик
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label="История статусов"
                      onClick={() => setHistoryTaskId(task.id)}
                    >
                      <HistoryIcon />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Проверить задачу ИИ"
                      disabled={analyze.isPending}
                      onClick={() => analyze.mutate(task.id)}
                    >
                      <SparklesIcon />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Удалить задачу"
                      disabled={deleteTask.isPending}
                      onClick={() => handleDelete(task.id, task.title)}
                    >
                      <Trash2Icon />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {isPaginationVisible && (
        <Pagination
          label="Страницы задач"
          total={data.total}
          hasPrev={pagination.hasPrev}
          hasNext={pagination.hasNext(data.total)}
          onPrev={pagination.toPrev}
          onNext={pagination.toNext}
        />
      )}

      {analyze.data && (
        <div
          className="bg-muted space-y-1 rounded-md p-3 text-sm"
          aria-live="polite"
        >
          <Badge
            variant={
              analyze.data.verdict === 'ok' ? 'secondary' : 'destructive'
            }
          >
            {analyze.data.verdict === 'ok' ? 'Замечаний нет' : 'Нужна проверка'}
          </Badge>
          {analyze.data.flags.map((flag) => (
            <p key={flag.code}>{flag.message}</p>
          ))}
          {analyze.data.advice && (
            <p className="text-muted-foreground">
              {analyze.data.advice.summary} {analyze.data.advice.suggestion}
            </p>
          )}
          <p className="text-muted-foreground text-xs">
            {analyze.data.source === 'rules+llm'
              ? 'Правила и модель.'
              : 'Только правила (модель недоступна).'}{' '}
            Решение остаётся за вами.
          </p>
        </div>
      )}

      {historyTask && (
        <TaskHistoryDialog
          taskId={historyTask.id}
          taskTitle={historyTask.title}
          open={historyTaskId !== null}
          onOpenChange={(open) => !open && setHistoryTaskId(null)}
        />
      )}
    </div>
  );
};
