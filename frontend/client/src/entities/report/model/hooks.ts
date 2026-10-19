import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { adminListReports } from '../api';
import { REPORT_KEYS } from './constants';
import type { AdminReportsParams } from './types';

export const useAdminReports = (params: AdminReportsParams = {}) =>
  useQuery({
    queryKey: REPORT_KEYS.ADMIN_LIST(params),
    queryFn: () => adminListReports(params),
    placeholderData: keepPreviousData,
    meta: { silent: true },
  });
