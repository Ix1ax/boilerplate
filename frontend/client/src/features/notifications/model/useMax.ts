import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { createMaxLink, getMaxStatus, unlinkMax } from '../api';

const KEY = ['notifications', 'max'] as const;

/** Канал MAX: статус, привязка по одноразовой ссылке и отвязка. */
export const useMax = () => {
  const client = useQueryClient();
  const status = useQuery({ queryKey: KEY, queryFn: getMaxStatus });
  const refresh = () => client.invalidateQueries({ queryKey: KEY });
  const link = useMutation({
    mutationFn: createMaxLink,
    // Ссылка открывает диалог с ботом; привязка завершится в MAX, статус обновим при возврате.
    onSuccess: ({ url }) => {
      window.open(url, '_blank', 'noopener,noreferrer');
      window.addEventListener('focus', () => void refresh(), { once: true });
    },
  });
  const unlink = useMutation({ mutationFn: unlinkMax, onSuccess: refresh });
  return { status: status.data, link, unlink };
};
