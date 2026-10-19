import { request } from '@/shared/api';

import type {
  AdminOrganizationsParams,
  Organization,
  OrganizationDetail,
  OrganizationPage,
  OrganizationValues,
} from '../model/types';

export const getMyOrganization = () =>
  request<Organization>({ url: '/my/organization' });
export const submitOrganization = (data: OrganizationValues) =>
  request<Organization>({ url: '/my/organization', method: 'POST', data });
export const updateMyOrganization = (data: OrganizationValues) =>
  request<Organization>({ url: '/my/organization', method: 'PUT', data });

export const adminListOrganizations = ({
  limit = 20,
  ...params
}: AdminOrganizationsParams = {}) =>
  request<OrganizationPage>({
    url: '/admin/organizations',
    params: { limit, ...params },
  });

export const adminGetOrganization = (id: string) =>
  request<OrganizationDetail>({ url: `/admin/organizations/${id}` });

export const approveOrganization = (id: string) =>
  request<Organization>({
    url: `/admin/organizations/${id}/approve`,
    method: 'POST',
  });

export const rejectOrganization = (id: string, reason: string) =>
  request<Organization>({
    url: `/admin/organizations/${id}/reject`,
    method: 'POST',
    data: { reason },
  });
