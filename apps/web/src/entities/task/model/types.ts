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

export type {
  Task as TaskDto,
  TaskPage as TaskPageDto,
} from '@starter/contracts';
