import { useQuery } from '@tanstack/react-query';

import { getMe } from '../api';
import { USER_KEYS } from './constants';

const STALE_TIME = 30_000;

export const useCurrentUser = () =>
  useQuery({
    queryKey: USER_KEYS.ME,
    queryFn: getMe,
    staleTime: STALE_TIME,
    retry: false,
    meta: { silent: true },
  });
