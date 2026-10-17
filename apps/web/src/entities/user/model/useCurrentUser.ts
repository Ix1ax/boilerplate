import { useQuery } from '@tanstack/react-query';

import { getMe } from '../api';
import { USER_KEYS } from './constants';

export const useCurrentUser = () => {
  return useQuery({
    queryKey: USER_KEYS.ME,
    queryFn: getMe,
    staleTime: 30_000,
    retry: false,
    meta: { silent: true },
  });
};
