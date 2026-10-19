import type { CompanyPlanValues, CompanyProposal } from '@starter/contracts';

import type { Camelize } from '@/shared/api';
import { request } from '@/shared/api';

import type { CompanyRequest } from '../model/types';
export type Proposal = Camelize<CompanyProposal>;
export const updateCompanyRequest = (id: string, data: CompanyPlanValues) =>
  request<CompanyRequest>({
    url: `/my/company-requests/${id}`,
    method: 'PATCH',
    data,
  });
export const getCompanyProposal = (id: string) =>
  request<{ proposal: Proposal | null }>({
    url: `/my/company-requests/${id}/proposal`,
  });
export const respondCompanyProposal = (
  id: string,
  version: string,
  interested: boolean,
) =>
  request<{ proposal: Proposal | null }>({
    url: `/my/company-requests/${id}/proposal/response`,
    method: 'POST',
    data: { version, interested },
  });
export const getCompanyDemand = () =>
  request<{ items: Proposal[] }>({ url: '/sports/company-demand' });
