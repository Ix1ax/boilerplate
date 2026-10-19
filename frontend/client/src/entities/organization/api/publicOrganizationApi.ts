import { request } from '@/shared/api';

import type { PublicOrganization } from '../model/types';

export const getOrganization = (id: string) =>
  request<PublicOrganization>({
    url: `/organizations/${encodeURIComponent(id)}`,
  });
