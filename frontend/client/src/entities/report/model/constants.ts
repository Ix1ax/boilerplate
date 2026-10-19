import type { AdminReportsParams } from './types';

export const REPORT_KEYS = {
  ALL: ['reports'] as const,
  ADMIN_LIST: (params: AdminReportsParams) =>
    ['reports', 'admin', 'list', params] as const,
};
