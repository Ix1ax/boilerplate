import { useQuery } from '@tanstack/react-query';

import { getTasks } from '../api';
import { TASK_KEYS } from './constants';
import type { TaskListParams } from './types';

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
