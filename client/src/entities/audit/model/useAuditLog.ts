import { useQuery } from '@tanstack/react-query';

import { getAuditLog } from '../api/auditApi';
import { AUDIT_KEYS } from './constants';
import type { AuditListParams } from './types';

export const useAuditLog = (params: AuditListParams = {}) =>
  useQuery({
    queryKey: AUDIT_KEYS.LIST(params),
    queryFn: () => getAuditLog(params),
  });
