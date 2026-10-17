import { useTasks } from '@/entities/task';

import { useOffsetPagination } from '@/shared/hooks';
import { Button, Skeleton } from '@/shared/ui';

import { TaskItem } from './TaskItem';

export const TaskList = () => {
  const pagination = useOffsetPagination();
  const { data, isPending, isError, refetch } = useTasks({
    offset: pagination.offset,
    limit: pagination.pageSize,
  });

  if (isPending) return <Skeleton className="h-24 w-full" />;
  if (isError)
    return (
      <div role="alert">
        Не удалось загрузить задачи.{' '}
        <Button onClick={() => void refetch()}>Повторить</Button>
      </div>
    );

  return (
    <div className="space-y-4">
      {data.items.length ? (
        <ul className="divide-y">
          {data.items.map((task) => (
            <TaskItem key={task.id} task={task} />
          ))}
        </ul>
      ) : (
        <p className="text-sm text-muted-foreground">
          Задач пока нет на этой странице
        </p>
      )}
      <nav aria-label="Страницы задач" className="flex items-center gap-3">
        <Button
          variant="outline"
          disabled={!pagination.hasPrev}
          onClick={pagination.toPrev}
        >
          Назад
        </Button>
        <span className="text-sm">Всего: {data.total}</span>
        <Button
          variant="outline"
          disabled={!pagination.hasNext(data.total)}
          onClick={pagination.toNext}
        >
          Далее
        </Button>
      </nav>
    </div>
  );
};
