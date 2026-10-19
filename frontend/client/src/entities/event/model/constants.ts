import type { EventListParams } from './types';

export const EVENT_KEYS = {
  ALL: ['events'] as const,
  LIST: (params: EventListParams) => ['events', 'list', params] as const,
  DETAIL: (id: string) => ['events', 'detail', id] as const,
  MY_PARTICIPATIONS: ['events', 'my', 'participations'] as const,
  MY_EVENTS: ['events', 'my', 'events'] as const,
};
