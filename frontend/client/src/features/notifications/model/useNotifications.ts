import { useEffect } from 'react';
import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';

import { EVENT_KEYS } from '@/entities/event';
import { ORGANIZATION_KEYS } from '@/entities/organization';

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
  const query = useInfiniteQuery({
    queryKey: KEY,
    initialPageParam: 0,
    queryFn: ({ pageParam }) => getNotifications(pageParam),
    getNextPageParam: (last) =>
      last.offset + last.items.length < last.total
        ? last.offset + last.limit
        : undefined,
    refetchInterval: 60_000,
    meta: { silent: true },
  });
  useEffect(() => {
    if (!query.dataUpdatedAt) return;
    void client.invalidateQueries({ queryKey: EVENT_KEYS.ALL });
    void client.invalidateQueries({ queryKey: ORGANIZATION_KEYS.ALL });
  }, [client, query.dataUpdatedAt]);
  return query;
};

export const useMarkRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id?: string) => (id ? markRead(id) : markAllRead()),
    onSuccess: () => client.invalidateQueries({ queryKey: KEY }),
  });
};
