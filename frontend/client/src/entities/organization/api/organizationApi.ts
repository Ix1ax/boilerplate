import { request } from '@/shared/api';

import type { Organization, OrganizationValues } from '../model/types';

export const getMyOrganization = () =>
  request<Organization>({ url: '/my/organization' });
export const submitOrganization = (data: OrganizationValues) =>
  request<Organization>({ url: '/my/organization', method: 'POST', data });
export const updateMyOrganization = (data: OrganizationValues) =>
  request<Organization>({ url: '/my/organization', method: 'PUT', data });
