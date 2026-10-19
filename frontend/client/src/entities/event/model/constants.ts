import type { AdminEventsParams, EventListParams } from './types';

export const EVENT_KEYS = {
  ALL: ['events'] as const,
  LIST: (params: EventListParams) => ['events', 'list', params] as const,
  DETAIL: (id: string) => ['events', 'detail', id] as const,
  MY_PARTICIPATIONS: ['events', 'my', 'participations'] as const,
  MY_EVENTS: ['events', 'my', 'events'] as const,
  ADMIN_LIST: (params: AdminEventsParams) =>
    ['events', 'admin', 'list', params] as const,
  ADMIN_HISTORY: (id: string) => ['events', 'admin', 'history', id] as const,
};
