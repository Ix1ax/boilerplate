import { keepPreviousData, useQuery } from '@tanstack/react-query';

import {
  getEvent,
  getEvents,
  getEventWeather,
  getMyEvents,
  getMyParticipations,
} from '../api';
import { EVENT_KEYS } from './constants';
import { type EventListParams } from './types';

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
