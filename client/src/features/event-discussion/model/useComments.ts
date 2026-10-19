import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { addComment, getComments } from '../api';

export const COMMENT_KEYS = {
  LIST: (eventId: string) => ['events', 'comments', eventId] as const,
};

export const useComments = (eventId: string) =>
  useQuery({
    queryKey: COMMENT_KEYS.LIST(eventId),
    queryFn: () => getComments(eventId),
    meta: { silent: true },
  });

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
