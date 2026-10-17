import { useMutation, useQueryClient } from '@tanstack/react-query';

import { logout } from '../api';

export const useLogout = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: logout,
    meta: { silent: true },
    onSettled: (data) => {
      queryClient.clear();
      if (data) window.location.assign(data.url);
    },
  });
};
