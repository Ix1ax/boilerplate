import { type TaskStats } from '@starter/contracts';
import { useQuery } from '@tanstack/react-query';

import { request } from '@/shared/api';

import { getTasks } from '../api';
import { TASK_KEYS } from './constants';
import { toTransitions } from './mappers';
import { type TaskHistoryPageDto, type TaskListParams } from './types';

export const useTaskHistory = (id: string, enabled = true) =>
  useQuery({
    queryKey: TASK_KEYS.HISTORY(id),
    queryFn: () =>
      request<TaskHistoryPageDto>({ url: `/tasks/${id}/history` }).then(
        toTransitions,
      ),
    enabled,
  });

export const useTaskStats = () =>
  useQuery({
    queryKey: ['admin', 'tasks', 'stats'],
    queryFn: () => request<TaskStats>({ url: '/admin/tasks/stats' }),
  });

export const useTasks = (params: TaskListParams = {}) => {
  return useQuery({
    queryKey: TASK_KEYS.LIST(params),
    queryFn: () => getTasks(params),
    // Пока очередь не проверила новую задачу, периодически обновляем список.
    refetchInterval: (query) =>
      query.state.data?.items.some((task) => task.moderation === null)
        ? 3000
        : false,
  });
};
