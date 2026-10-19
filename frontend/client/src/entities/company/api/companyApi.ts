import { request } from '@/shared/api';

import type {
  CompanyRequest,
  CompanyRequestValues,
  SportEventPage,
} from '../model/types';
import type { ListMyCompanyRequestsResponse } from './types';

export const listMyCompanyRequests = () =>
  request<ListMyCompanyRequestsResponse>({ url: '/my/company-requests' });

export const createCompanyRequest = (data: CompanyRequestValues) =>
  request<CompanyRequest>({
    url: '/my/company-requests',
    method: 'POST',
    data,
  });

export const closeCompanyRequest = (id: string) =>
  request<void>({ url: `/my/company-requests/${id}`, method: 'DELETE' });

export const listCompanyMatches = (id: string, offset = 0, limit = 12) =>
  request<SportEventPage>({
    url: `/my/company-requests/${id}/matches`,
    params: { limit, offset },
  });
