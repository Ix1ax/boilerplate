import { useQuery } from '@tanstack/react-query';

import { getEvent, getEventWeather } from '../api';
import { EVENT_KEYS } from './constants';

export const useEvent = (id: string) =>
  useQuery({
    queryKey: EVENT_KEYS.DETAIL(id),
    queryFn: () => getEvent(id),
    meta: { silent: true },
  });

export const useEventWeather = (id: string, enabled: boolean) =>
  useQuery({
    queryKey: [...EVENT_KEYS.DETAIL(id), 'weather'],
    queryFn: () => getEventWeather(id),
    enabled,
    staleTime: 10 * 60_000,
    meta: { silent: true },
  });
