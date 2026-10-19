import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { demoLogin, getDemoAccounts } from '../api';

export const useDemoAccounts = () =>
  useQuery({
    queryKey: ['auth', 'demo'],
    queryFn: getDemoAccounts,
    staleTime: Infinity,
    meta: { silent: true },
  });

export const useDemoLogin = (next: string | ((key: string) => string)) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: demoLogin,
    meta: { silent: true },
    onSuccess: (_, key) => {
      // Сессия заменена: данные прежнего пользователя не должны мелькнуть.
      queryClient.clear();
      window.location.assign(typeof next === 'function' ? next(key) : next);
    },
  });
};
