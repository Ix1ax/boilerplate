import { request } from '@/shared/api';

import type { AdminReportsParams, Report, ReportPage } from '../model/types';

export const adminListReports = ({
  limit = 20,
  ...params
}: AdminReportsParams = {}) =>
  request<ReportPage>({ url: '/admin/reports', params: { limit, ...params } });

export const dismissReport = (id: string, resolution: string) =>
  request<Report>({
    url: `/admin/reports/${id}/dismiss`,
    method: 'POST',
    data: { resolution },
  });
