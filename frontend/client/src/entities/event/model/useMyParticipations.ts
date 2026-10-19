import { useQuery } from '@tanstack/react-query';

import { getMyEvents, getMyParticipations } from '../api';
import { EVENT_KEYS } from './constants';

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
