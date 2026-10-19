import type { AuditListParams } from './types';

export const AUDIT_KEYS = {
  ALL: ['audit'] as const,
  LIST: (params: AuditListParams = {}) => ['audit', 'list', params] as const,
};
