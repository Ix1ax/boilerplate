import { useEffect } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { useCurrentUser } from '@/entities/user';

import { useWsClient } from '@/shared/ws';

import { addComment, getComments } from '../api';

export const COMMENT_KEYS = {
  LIST: (eventId: string) => ['events', 'comments', eventId] as const,
};

export const useComments = (eventId: string) => {
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  const ws = useWsClient();
  useEffect(() => {
    if (!user) return;
    const channel = `room:event:${eventId}`;
    ws.connect();
    ws.send({ type: 'subscribe', payload: { channel } });
    const off = ws.on('message', ({ payload }) => {
      if (payload.channel === channel)
        void client.invalidateQueries({ queryKey: COMMENT_KEYS.LIST(eventId) });
    });
    const statusOff = ws.onStatus((status) => {
      if (status === 'open')
        void client.invalidateQueries({ queryKey: COMMENT_KEYS.LIST(eventId) });
    });
    return () => {
      off();
      statusOff();
      ws.send({ type: 'unsubscribe', payload: { channel } });
    };
  }, [eventId, ws, client, user]);
  return useQuery({
    queryKey: COMMENT_KEYS.LIST(eventId),
    queryFn: () => getComments(eventId),
    meta: { silent: true },
    refetchInterval: 30_000,
  });
};

export const useAddComment = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: addComment,
    // Ошибку показываем под полем: текст остаётся, можно отправить ещё раз.
    meta: { silent: true },
    onSuccess: (_comment, { eventId }) =>
      queryClient.invalidateQueries({ queryKey: COMMENT_KEYS.LIST(eventId) }),
  });
};
