import {
  keepPreviousData,
  useInfiniteQuery,
  useQuery,
} from '@tanstack/react-query';

import {
  adminEventHistory,
  adminListEvents,
  getEvent,
  getEvents,
  getEventWeather,
  getMyEvents,
  getMyParticipations,
} from '../api';
import { EVENT_KEYS } from './constants';
import { type AdminEventsParams, type EventListParams } from './types';

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

export const useEvents = (
  params: EventListParams = {},
  options: { enabled?: boolean } = {},
) =>
  useQuery({
    queryKey: EVENT_KEYS.LIST(params),
    queryFn: () => getEvents(params),
    // Фильтры меняются часто: старый список остаётся на экране, пока грузится новый.
    placeholderData: keepPreviousData,
    enabled: options.enabled ?? true,
    meta: { silent: true },
  });

export const useMyParticipations = () =>
  useQuery({
    queryKey: EVENT_KEYS.MY_PARTICIPATIONS,
    queryFn: getMyParticipations,
    meta: { silent: true },
  });

export const useMyEvents = () =>
  useQuery({
    queryKey: EVENT_KEYS.MY_EVENTS,
    queryFn: getMyEvents,
    meta: { silent: true },
  });

export const useAdminEvents = (params: AdminEventsParams = {}) =>
  useQuery({
    queryKey: EVENT_KEYS.ADMIN_LIST(params),
    queryFn: () => adminListEvents(params),
    placeholderData: keepPreviousData,
    meta: { silent: true },
  });

export const useAdminEventHistory = (id: string, enabled: boolean) =>
  useQuery({
    queryKey: EVENT_KEYS.ADMIN_HISTORY(id),
    queryFn: () => adminEventHistory(id),
    enabled,
    meta: { silent: true },
  });

export const useEventPages = (params: EventListParams = {}) =>
  useInfiniteQuery({
    queryKey: [...EVENT_KEYS.LIST(params), 'pages'],
    initialPageParam: 0,
    queryFn: ({ pageParam }) =>
      getEvents({ ...params, limit: 24, offset: pageParam }),
    getNextPageParam: (last) =>
      last.offset + last.items.length < last.total
        ? last.offset + last.items.length
        : undefined,
    meta: { silent: true },
  });
