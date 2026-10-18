import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { getNotifications, markAllRead, markRead } from '../api';

const KEY = ['notifications'] as const;

/** Уведомления приходят асинхронно (через RabbitMQ), поэтому список периодически обновляется. */
export const useNotifications = () =>
  useQuery({
    queryKey: KEY,
    queryFn: getNotifications,
    refetchInterval: 15_000,
  });

export const useMarkRead = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id?: string) => (id ? markRead(id) : markAllRead()),
    onSuccess: () => client.invalidateQueries({ queryKey: KEY }),
  });
};
