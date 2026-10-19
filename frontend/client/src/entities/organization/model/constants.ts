import type { AdminOrganizationsParams } from './types';

export const ORGANIZATION_KEYS = {
  DETAIL: (id: string) => ['organizations', 'detail', id] as const,
  ALL: ['organizations'] as const,
  MY: ['organizations', 'my'] as const,
  ADMIN_LIST: (params: AdminOrganizationsParams) =>
    ['organizations', 'admin', 'list', params] as const,
  ADMIN_DETAIL: (id: string) =>
    ['organizations', 'admin', 'detail', id] as const,
};
