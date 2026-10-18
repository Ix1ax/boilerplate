import type { TaskStats } from '@starter/contracts';
import { useQuery } from '@tanstack/react-query';

import { request } from '@/shared/api';

export const useTaskStats = () =>
  useQuery({
    queryKey: ['admin', 'tasks', 'stats'],
    queryFn: () => request<TaskStats>({ url: '/admin/tasks/stats' }),
  });
