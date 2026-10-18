import type { TaskListParams } from './types';

export const TASK_KEYS = {
  ALL: ['tasks'] as const,
  LIST: (params: TaskListParams = {}) => ['tasks', 'list', params] as const,
  HISTORY: (id: string) => ['tasks', 'history', id] as const,
};
