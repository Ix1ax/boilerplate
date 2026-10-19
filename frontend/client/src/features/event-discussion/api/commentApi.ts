import type { EventComment as CommentDto } from '@starter/contracts';

import { type Camelize, request } from '@/shared/api';
export type EventComment = Camelize<CommentDto>;

export const getComments = (eventId: string) =>
  request<{ items: EventComment[] }>({
    url: `/events/${eventId}/comments`,
  }).then((data) => data.items);

export type AddCommentPayload = {
  eventId: string;
  body: string;
  parentId?: string | null;
};

export const addComment = ({ eventId, body, parentId }: AddCommentPayload) =>
  request<EventComment>({
    url: `/events/${eventId}/comments`,
    method: 'POST',
    data: { body, parent_id: parentId ?? null },
  });

export const deleteComment = (eventId: string, id: string) =>
  request<EventComment>({
    url: `/events/${eventId}/comments/${id}`,
    method: 'DELETE',
  });
