import type { ProfileSettingsValues } from '@starter/contracts';

import { request } from '@/shared/api';

import type { Profile } from '../model';

export const getProfile = (id: string) =>
  request<Profile>({ url: `/profiles/${encodeURIComponent(id)}` });

export const getMyProfile = () => request<Profile>({ url: '/my/profile' });
export const updateMyProfile = (data: ProfileSettingsValues) =>
  request<Profile>({ url: '/my/profile', method: 'PATCH', data });
