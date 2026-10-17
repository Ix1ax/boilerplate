import { useNavigate } from 'react-router';
import { useMutation, useQueryClient } from '@tanstack/react-query';

import { clearTokens } from '@/entities/user';

import { STATIC_ROUTES } from '@/shared/constants';
import { wsClient } from '@/shared/ws';

import { logout } from '../api';

export const useLogout = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: logout,
    meta: { silent: true },
    onSettled: () => {
      clearTokens();

      wsClient.close();
      queryClient.clear();

      navigate(STATIC_ROUTES.LOGIN, { replace: true });
    },
  });
};
