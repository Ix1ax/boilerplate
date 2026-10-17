import { useLocation, useNavigate } from 'react-router';
import { useMutation, useQueryClient } from '@tanstack/react-query';

import { setTokens, USER_KEYS } from '@/entities/user';

import { STATIC_ROUTES } from '@/shared/constants';
import { wsClient } from '@/shared/ws';

import { login } from '../api';

type SavedLocation = {
  pathname?: string;
  search?: string;
  hash?: string;
};

type LocationState = {
  from?: SavedLocation;
} | null;

const toPath = (from?: SavedLocation) =>
  from?.pathname
    ? `${from.pathname}${from.search ?? ''}${from.hash ?? ''}`
    : undefined;

export const useLogin = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();

  return useMutation({
    mutationFn: login,
    onSuccess: async (tokens) => {
      setTokens(tokens);
      wsClient.reconnect();

      await queryClient.invalidateQueries({ queryKey: USER_KEYS.ME });
      const from = toPath((location.state as LocationState)?.from);

      navigate(from ?? STATIC_ROUTES.DASHBOARD, {
        replace: true,
      });
    },
  });
};
