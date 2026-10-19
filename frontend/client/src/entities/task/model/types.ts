export type TaskStatus = 'draft' | 'todo' | 'in_progress' | 'done' | 'archived';

export type Task = {
  id: string;
  title: string;
  status: TaskStatus;
  createdAt: string;
  /** null, пока асинхронная проверка не выполнена. */
  moderation: { verdict: 'ok' | 'needs_review'; source: string } | null;
};

export type TaskPage = {
  items: Task[];
  total: number;
  limit: number;
  offset: number;
};

export type TaskListParams = {
  limit?: number;
  offset?: number;
  q?: string;
  status?: TaskStatus;
  sort?: string;
};

export type TaskTransition = {
  from: TaskStatus;
  to: TaskStatus;
  actor: string;
  comment: string;
  createdAt: string;
};

// DTO с проволоки — форма ответа Go API из общего контракта, не доменная модель выше.
export type {
  Task as TaskDto,
  TaskHistoryPage as TaskHistoryPageDto,
  TaskPage as TaskPageDto,
  TaskTransition as TaskTransitionDto,
} from '@starter/contracts';
