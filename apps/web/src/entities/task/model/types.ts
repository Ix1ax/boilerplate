export type Task = {
  id: string;
  title: string;
  completed: boolean;
  createdAt: string;
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
};

// DTO с проволоки — форма ответа Go API из общего контракта, не доменная модель выше.
export type {
  Task as TaskDto,
  TaskPage as TaskPageDto,
} from '@starter/contracts';
