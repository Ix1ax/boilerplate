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

export type TaskDto = {
  id: string;
  title: string;
  completed: boolean;
  created_at: string;
};

export type TaskPageDto = {
  items: TaskDto[];
  total: number;
  limit: number;
  offset: number;
};
