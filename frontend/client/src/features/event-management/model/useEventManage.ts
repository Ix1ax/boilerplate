import { useState } from 'react';
import { reasonSchema } from '@starter/contracts';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  approveParticipant,
  cancelEvent,
  EVENT_KEYS,
  listParticipants,
  rejectParticipant,
  remindParticipants,
  useEvent,
} from '@/entities/event';

export const useEventManage = (
  eventId: string,
  onConfirmCancel: () => Promise<boolean>,
) => {
  const cache = useQueryClient();
  const event = useEvent(eventId);
  const participants = useQuery({
    queryKey: [...EVENT_KEYS.DETAIL(eventId), 'participants'],
    queryFn: () => listParticipants(eventId),
    enabled: Boolean(event.data?.is_organizer),
    meta: { silent: true },
  });
  const [feedback, setFeedback] = useState('');
  const [reason, setReason] = useState('');
  const [reasonError, setReasonError] = useState('');
  const action = useMutation({
    mutationFn: async (task: {
      kind: 'approve' | 'reject' | 'remind' | 'cancel';
      participantId?: string;
      reason?: string;
    }) => {
      if (task.kind === 'approve') {
        await approveParticipant(eventId, task.participantId!);
        return 'Заявка принята.';
      }
      if (task.kind === 'reject') {
        await rejectParticipant(
          eventId,
          task.participantId!,
          task.reason ?? '',
        );
        return 'Заявка отклонена.';
      }
      if (task.kind === 'cancel') {
        await cancelEvent(eventId, task.reason!);
        return 'Встреча отменена. Участники получат уведомление.';
      }
      const result = await remindParticipants(eventId);
      return `Напоминание отправлено участникам: ${result.sent}.`;
    },
    onSuccess: async (message) => {
      setFeedback(message);
      await cache.invalidateQueries({ queryKey: EVENT_KEYS.ALL });
    },
    onError: () => setFeedback(''),
  });
  const cancel = async () => {
    const parsed = reasonSchema.safeParse({ reason });
    if (!parsed.success) {
      setReasonError(parsed.error.issues[0].message);
      return;
    }
    setReasonError('');
    const accepted = await onConfirmCancel();
    if (accepted)
      action.mutate({
        kind: 'cancel',
        reason: parsed.data.reason,
      });
  };
  return {
    eventId,
    event,
    participants,
    feedback,
    reason,
    setReason,
    reasonError,
    action,
    cancel,
  };
};
