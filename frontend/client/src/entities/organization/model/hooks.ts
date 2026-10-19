import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { ApiError } from '@/shared/api';

import {
  adminGetOrganization,
  adminListOrganizations,
  getMyOrganization,
  getOrganization,
} from '../api';
import { ORGANIZATION_KEYS } from './constants';
import type { AdminOrganizationsParams } from './types';

export const useMyOrganization = () =>
  useQuery({
    queryKey: ORGANIZATION_KEYS.MY,
    queryFn: async () => {
      try {
        return await getMyOrganization();
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return null;
        throw error;
      }
    },
    meta: { silent: true },
  });

export const useOrganizationProfile = (id: string) =>
  useQuery({
    queryKey: ORGANIZATION_KEYS.DETAIL(id),
    queryFn: () => getOrganization(id),
    enabled: Boolean(id),
    meta: { silent: true },
  });

export const useAdminOrganizations = (params: AdminOrganizationsParams = {}) =>
  useQuery({
    queryKey: ORGANIZATION_KEYS.ADMIN_LIST(params),
    queryFn: () => adminListOrganizations(params),
    placeholderData: keepPreviousData,
    meta: { silent: true },
  });

export const useAdminOrganization = (id: string) =>
  useQuery({
    queryKey: ORGANIZATION_KEYS.ADMIN_DETAIL(id),
    queryFn: () => adminGetOrganization(id),
    enabled: Boolean(id),
    meta: { silent: true },
  });
