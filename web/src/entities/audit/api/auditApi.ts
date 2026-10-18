import { request } from '@/shared/api';

import { toAuditPage } from '../model/mappers';
import type { AuditListParams, AuditPageDto } from '../model/types';

export const getAuditLog = ({
  limit = 20,
  offset = 0,
  q,
  actor,
  method,
  status,
  requestId,
  sort,
}: AuditListParams = {}) =>
  request<AuditPageDto>({
    url: '/admin/audit',
    params: {
      limit,
      offset,
      q,
      actor,
      method,
      status,
      request_id: requestId,
      sort,
    },
  }).then(toAuditPage);
