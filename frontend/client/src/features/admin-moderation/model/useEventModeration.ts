import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  deleteEvent,
  EVENT_KEYS,
  restoreEvent,
  unpublishEvent,
} from '@/entities/event';

export const useUnpublishEvent = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      unpublishEvent(id, reason),
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin'] });
      void queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: EVENT_KEYS.ALL });
    },
  });
};

export const useDeleteEvent = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      deleteEvent(id, reason),
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin'] });
      void queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: EVENT_KEYS.ALL });
    },
  });
};

export const useRestoreEvent = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => restoreEvent(id),
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin'] });
      void queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: EVENT_KEYS.ALL });
    },
  });
};
