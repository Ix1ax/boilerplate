import { request } from '@/shared/api';

import type {
  Event,
  EventListParams,
  EventPage,
  EventWeather,
} from '../model/types';
import type { GetMyParticipationsResponse } from './types';

export const getEvents = ({ limit = 24, ...params }: EventListParams = {}) =>
  request<EventPage>({ url: '/events', params: { limit, ...params } });

export const getEvent = (id: string) =>
  request<Event>({ url: `/events/${id}` });

export const getMyParticipations = () =>
  request<GetMyParticipationsResponse>({ url: '/my/participations' }).then(
    (data) => data.items,
  );

export const getMyEvents = () => request<EventPage>({ url: '/my/events' });

export const getEventWeather = (id: string) =>
  request<EventWeather>({ url: `/events/${id}/weather` });
