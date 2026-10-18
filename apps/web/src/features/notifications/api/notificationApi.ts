import type { NotificationPage } from '@starter/contracts';

import { request } from '@/shared/api';

export const getNotifications = () =>
  request<NotificationPage>({ url: '/notifications', params: { limit: 20 } });

export const markRead = (id: string) =>
  request<void>({ url: `/notifications/${id}/read`, method: 'POST' });

export const markAllRead = () =>
  request<void>({ url: '/notifications/read-all', method: 'POST' });
