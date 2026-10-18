import { cn } from 'cn';
import { SparklesIcon, Trash2Icon } from 'lucide-react';

import { useConfirm } from '@/features/confirm';

import type { Task } from '@/entities/task';

import { Badge, Button, Checkbox } from '@/shared/ui';

import { useAnalyzeTask, useDeleteTask, useToggleTask } from '../model';

interface TaskItemProps {
  task: Task;
}

export const TaskItem = ({ task }: TaskItemProps) => {
  const toggleTask = useToggleTask();
  const deleteTask = useDeleteTask();
  const analyze = useAnalyzeTask();
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
    <li className="py-2">
      <div className="flex items-center gap-3">
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
            'min-w-0 flex-1 truncate text-sm',
            task.completed && 'text-muted-foreground line-through',
          )}
          title={task.title}
        >
          {task.title}
        </span>
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
          onClick={handleDelete}
        >
          <Trash2Icon />
        </Button>
      </div>
      {analyze.data && (
        <div
          className="bg-muted mt-2 space-y-1 rounded-md p-3 text-sm"
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
    </li>
  );
};
