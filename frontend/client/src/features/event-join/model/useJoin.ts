import { useMutation, useQueryClient } from '@tanstack/react-query';

import { EVENT_KEYS } from '@/entities/event';

import { joinEvent, leaveEvent } from '../api';

// Запись меняет и карточку, и «Мой спорт», и счётчики в каталоге — обновляем всё про события.
export const useJoinEvent = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: joinEvent,
    meta: { silent: true },
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: EVENT_KEYS.ALL }),
  });
};

export const useLeaveEvent = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: leaveEvent,
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: EVENT_KEYS.ALL }),
  });
};
