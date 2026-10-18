import { useEffect } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { env } from '@/shared/config';

import { getNotifications, markAllRead, markRead } from '../api';

const KEY = ['notifications'] as const;

/**
 * Новые уведомления приходят по SSE (EventSource сам переподключается); редкий опрос —
 * запасной путь, если поток недоступен (например, прокси без поддержки SSE).
 */
export const useNotifications = () => {
  const client = useQueryClient();
  useEffect(() => {
    const source = new EventSource(`${env.VITE_API_URL}/notifications/stream`);
    source.addEventListener('notification', () => {
      void client.invalidateQueries({ queryKey: KEY });
    });
    return () => source.close();
  }, [client]);
  return useQuery({
    queryKey: KEY,
    queryFn: getNotifications,
    refetchInterval: 60_000,
  });
};

export const useMarkRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id?: string) => (id ? markRead(id) : markAllRead()),
    onSuccess: () => client.invalidateQueries({ queryKey: KEY }),
  });
};
