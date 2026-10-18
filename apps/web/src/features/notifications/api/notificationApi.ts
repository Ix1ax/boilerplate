import type { MaxStatus, NotificationPage } from '@starter/contracts';

import { request } from '@/shared/api';

export const getNotifications = () =>
  request<NotificationPage>({ url: '/notifications', params: { limit: 20 } });

export const markRead = (id: string) =>
  request<void>({ url: `/notifications/${id}/read`, method: 'POST' });

export const markAllRead = () =>
  request<void>({ url: '/notifications/read-all', method: 'POST' });

export const getMaxStatus = () =>
  request<MaxStatus>({ url: '/notifications/max' });

export const createMaxLink = () =>
  request<{ url: string }>({ url: '/notifications/max/link', method: 'POST' });

export const unlinkMax = () =>
  request<void>({ url: '/notifications/max', method: 'DELETE' });
