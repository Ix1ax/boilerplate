export const SPORT_KEYS = {
  ALL: ['sports'] as const,
  LIST: ['sports', 'list'] as const,
  DEMAND: (params: object) => ['sports', 'demand', params] as const,
};
