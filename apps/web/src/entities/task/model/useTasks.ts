import { useQuery } from '@tanstack/react-query';

import { getTasks } from '../api';
import { TASK_KEYS } from './constants';
import type { TaskListParams } from './types';

export const useTasks = (params: TaskListParams = {}) => {
  return useQuery({
    queryKey: TASK_KEYS.LIST(params),
    queryFn: () => getTasks(params),
  });
};
