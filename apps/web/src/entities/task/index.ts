import type { CreateTask, Task, TaskPage } from '@starter/contracts';
import { api } from '@/shared/api/client';
export const tasksApi = {
  list: (offset = 0, signal?: AbortSignal) =>
    api<TaskPage>(`/tasks?limit=10&offset=${offset}`, { signal }),
  create: (input: CreateTask) =>
    api<Task>('/tasks', { method: 'POST', body: JSON.stringify(input) }),
  complete: (id: string, completed: boolean) =>
    api<Task>(`/tasks/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ completed }),
    }),
  remove: (id: string) => api<void>(`/tasks/${id}`, { method: 'DELETE' }),
};
