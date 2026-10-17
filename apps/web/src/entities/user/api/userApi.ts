import { request } from '@/shared/api';

import type { User } from '../model';
export const getMe = () =>
  request<User | null>({ url: '/me' }).catch((error) => {
    if (error.status === 401) return null;
    throw error;
  });
