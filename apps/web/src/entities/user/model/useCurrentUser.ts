import { useQuery } from '@tanstack/react-query';

import { getMe } from '../api';
import { hasSession } from '../lib';
import { USER_KEYS } from './constants';

export const useCurrentUser = () => {
  return useQuery({
    queryKey: USER_KEYS.ME,
    queryFn: getMe,
    enabled: hasSession(),
    staleTime: Infinity,
    retry: false,
    meta: { silent: true },
  });
}
