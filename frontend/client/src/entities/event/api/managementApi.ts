import { request } from '@/shared/api';

import type {
  AdminEventsParams,
  Event,
  EventInput,
  EventPage,
  EventPatch,
  HistoryEntry,
  Participant,
} from '../model/types';
import type {
  ListParticipantsResponse,
  RemindParticipantsResponse,
} from './types';

export const createEvent = (data: EventInput) =>
  request<Event>({ url: '/events', method: 'POST', data });

export const listParticipants = (id: string) =>
  request<ListParticipantsResponse>({ url: `/events/${id}/participants` }).then(
    (data) => data.items,
  );

export const approveParticipant = (id: string, participantId: string) =>
  request<Participant>({
    url: `/events/${id}/participants/${participantId}/approve`,
    method: 'POST',
  });

export const rejectParticipant = (
  id: string,
  participantId: string,
  reason: string,
) =>
  request<Participant>({
    url: `/events/${id}/participants/${participantId}/reject`,
    method: 'POST',
    data: { reason },
  });

export const remindParticipants = (id: string) =>
  request<RemindParticipantsResponse>({
    url: `/events/${id}/remind`,
    method: 'POST',
  });

export const cancelEvent = (id: string, reason: string) =>
  request<Event>({
    url: `/events/${id}/cancel`,
    method: 'POST',
    data: { reason },
  });

export const updateEvent = (id: string, data: EventPatch) =>
  request<Event>({ url: `/events/${id}`, method: 'PATCH', data });

export const adminListEvents = ({
  limit = 20,
  ...params
}: AdminEventsParams = {}) =>
  request<EventPage>({ url: '/admin/events', params: { limit, ...params } });

export const unpublishEvent = (id: string, reason: string) =>
  request<Event>({
    url: `/admin/events/${id}/unpublish`,
    method: 'POST',
    data: { reason },
  });

export const deleteEvent = (id: string, reason: string) =>
  request<Event>({
    url: `/admin/events/${id}/delete`,
    method: 'POST',
    data: { reason },
  });

export const restoreEvent = (id: string) =>
  request<Event>({ url: `/admin/events/${id}/restore`, method: 'POST' });

export const adminEventHistory = (id: string) =>
  request<{ items: HistoryEntry[] }>({
    url: `/admin/events/${id}/history`,
  }).then((data) => data.items);
