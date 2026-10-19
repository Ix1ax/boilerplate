import { request } from '@/shared/api';

import type { Profile } from '../model';

export const getProfile = (id: string) =>
  request<Profile>({ url: `/profiles/${encodeURIComponent(id)}` });
