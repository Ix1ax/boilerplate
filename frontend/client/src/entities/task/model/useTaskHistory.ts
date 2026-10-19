import { useQuery } from '@tanstack/react-query';

import { request } from '@/shared/api';

import { TASK_KEYS } from './constants';
import { toTransitions } from './mappers';
import type { TaskHistoryPageDto } from './types';

export const useTaskHistory = (id: string, enabled = true) =>
  useQuery({
    queryKey: TASK_KEYS.HISTORY(id),
    queryFn: () =>
      request<TaskHistoryPageDto>({ url: `/tasks/${id}/history` }).then(
        toTransitions,
      ),
    enabled,
  });
