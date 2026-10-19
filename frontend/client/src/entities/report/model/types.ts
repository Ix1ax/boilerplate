import type { Report as ReportDto, ReportPage as ReportPageDto } from '@starter/contracts';

import type { Camelize } from '@/shared/api';

export type Report = Camelize<ReportDto>;
export type ReportPage = Camelize<ReportPageDto>;

export type AdminReportsParams = {
  limit?: number;
  offset?: number;
  status?: 'open' | 'resolved' | 'dismissed';
  target_type?: 'event' | 'comment';
};
