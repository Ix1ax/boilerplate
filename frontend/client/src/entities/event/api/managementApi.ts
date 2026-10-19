import { request } from '@/shared/api';

import type { Event, EventInput, Participant } from '../model/types';
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
