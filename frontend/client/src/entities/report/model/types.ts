export type { Report, ReportPage } from '@starter/contracts';

export type AdminReportsParams = {
  limit?: number;
  offset?: number;
  status?: 'open' | 'resolved' | 'dismissed';
  target_type?: 'event' | 'comment';
};
