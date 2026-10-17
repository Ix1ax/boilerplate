import { useTasks } from '@/entities/task';

import { Skeleton } from '@/shared/ui';

import { TaskItem } from './TaskItem';

export const TaskList = () => {
  const { data, isPending } = useTasks();

  if (isPending) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-full" />
      </div>
    );
  }

  if (!data?.items.length) {
    return <p className="text-sm text-muted-foreground">Задач пока нет</p>;
  }

  return (
    <ul className="divide-y">
      {data.items.map((task) => (
        <TaskItem key={task.id} task={task} />
      ))}
    </ul>
  );
};
