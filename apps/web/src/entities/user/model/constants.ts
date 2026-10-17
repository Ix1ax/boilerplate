export const USER_ROLES = ['admin', 'organizer', 'participant'] as const;

export const USER_KEYS = {
  ALL: ['users'] as const,
  ME: ['users', 'me'] as const,
  LIST: () => ['users', 'list'] as const,
  DETAIL: (id: string) => ['users', 'detail', id] as const,
};
