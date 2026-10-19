import type { MyParticipation } from '@/entities/event';

import { request } from '@/shared/api';

export type JoinPayload = {
  eventId: string;
  message?: string;
  firstTime?: boolean;
};

export const joinEvent = ({ eventId, message, firstTime }: JoinPayload) =>
  request<MyParticipation>({
    url: `/events/${eventId}/join`,
    method: 'POST',
    data: { message: message || undefined, first_time: firstTime ?? false },
  });

export const leaveEvent = (eventId: string) =>
  request<void>({ url: `/events/${eventId}/join`, method: 'DELETE' });
