import type { TaskStatus } from './types';

/**
 * Зеркало конечного автомата статусов из apps/tasks/internal/tasks/tasks.go
 * (statusWorkflow): меняйте синхронно с бэкендом, иначе меню предложит переход,
 * который сервер отвергнет 409-м.
 */
export const STATUS_TRANSITIONS: Record<TaskStatus, TaskStatus[]> = {
  draft: ['todo'],
  todo: ['in_progress', 'archived'],
  in_progress: ['done', 'todo'],
  done: ['archived', 'in_progress'],
  archived: ['todo'],
};

/** Переходы, требующие обязательного комментария (см. RequireComment на бэкенде). */
export const STATUS_REQUIRES_COMMENT: Partial<Record<TaskStatus, boolean>> = {
  archived: true,
};

export const STATUS_META: Record<
  TaskStatus,
  {
    label: string;
    variant: 'default' | 'secondary' | 'outline';
    className?: string;
  }
> = {
  draft: { label: 'Черновик', variant: 'outline' },
  todo: { label: 'К выполнению', variant: 'secondary' },
  in_progress: { label: 'В работе', variant: 'default' },
  done: {
    label: 'Готово',
    variant: 'secondary',
    className:
      'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300',
  },
  archived: {
    label: 'В архиве',
    variant: 'secondary',
    className: 'bg-muted text-muted-foreground',
  },
};

export const STATUS_ORDER: TaskStatus[] = [
  'draft',
  'todo',
  'in_progress',
  'done',
  'archived',
];
