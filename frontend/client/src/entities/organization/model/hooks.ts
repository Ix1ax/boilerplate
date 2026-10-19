import { useQuery } from '@tanstack/react-query';

import { ApiError } from '@/shared/api';

import { getMyOrganization, getOrganization } from '../api';
import { ORGANIZATION_KEYS } from './constants';

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
