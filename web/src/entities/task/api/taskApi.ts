import { request } from '@/shared/api';

import { toTaskPage } from '../model/mappers';
import type { TaskListParams, TaskPageDto } from '../model/types';

export const getTasks = ({
  limit = 20,
  offset = 0,
  q,
  status,
  sort,
}: TaskListParams = {}) =>
  request<TaskPageDto>({
    url: '/tasks',
    params: { limit, offset, q, status, sort },
  }).then(toTaskPage);
