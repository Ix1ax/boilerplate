import { cn } from 'cn';
import { Trash2Icon } from 'lucide-react';

import { useConfirm } from '@/features/confirm';

import type { Task } from '@/entities/task';

import { Button, Checkbox } from '@/shared/ui';

import { useDeleteTask, useToggleTask } from '../model';

interface TaskItemProps {
  task: Task;
}

export const TaskItem = ({ task }: TaskItemProps) => {
  const toggleTask = useToggleTask();
  const deleteTask = useDeleteTask();
  const confirm = useConfirm();

  const handleDelete = async () => {
    const ok = await confirm({
      title: 'Удалить задачу?',
      description: task.title,
      destructive: true,
    });
    if (ok) deleteTask.mutate(task.id);
  };

  return (
    <li className="flex items-center gap-3 py-2">
      <Checkbox
        checked={task.completed}
        disabled={toggleTask.isPending}
        aria-label={task.title}
        onCheckedChange={(checked) =>
          toggleTask.mutate({ id: task.id, completed: checked === true })
        }
      />
      <span
        className={cn(
          'flex-1 text-sm',
          task.completed && 'text-muted-foreground line-through',
        )}
      >
        {task.title}
      </span>
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label="Удалить задачу"
        disabled={deleteTask.isPending}
        onClick={handleDelete}
      >
        <Trash2Icon />
      </Button>
    </li>
  );
};
