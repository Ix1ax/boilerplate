import { useMutation, useQueryClient } from '@tanstack/react-query';

import { logout } from '../api';
export const useLogout = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: logout,
    onSuccess: ({ url }) => {
      queryClient.clear();
      window.location.assign(url);
    },
  });
};
