export const ORGANIZATION_KEYS = {
  DETAIL: (id: string) => ['organizations', 'detail', id] as const,
  ALL: ['organizations'] as const,
  MY: ['organizations', 'my'] as const,
};
