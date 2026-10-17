import { type Task, type TaskDto, toTask } from '@/entities/task';

import { request } from '@/shared/api';

export const createTask = (title: string) =>
  request<TaskDto>({ url: '/tasks', method: 'POST', data: { title } }).then(
    toTask,
  );

export const updateTask = (id: string, completed: boolean): Promise<Task> =>
  request<TaskDto>({
    url: `/tasks/${id}`,
    method: 'PATCH',
    data: { completed },
  }).then(toTask);

export const deleteTask = (id: string) =>
  request<void>({ url: `/tasks/${id}`, method: 'DELETE' });
