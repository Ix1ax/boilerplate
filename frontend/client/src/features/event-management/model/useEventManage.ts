import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
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

import { request } from '@/shared/api';

export const useEventManage = (
  eventId: string,
  onConfirmCancel: () => Promise<boolean>,
) => {
  const cache = useQueryClient();
  const event = useEvent(eventId);
  const participants = useQuery({
    queryKey: [...EVENT_KEYS.DETAIL(eventId), 'participants'],
    queryFn: () => listParticipants(eventId),
    enabled: Boolean(event.data?.isOrganizer),
    meta: { silent: true },
  });
  const [feedback, setFeedback] = useState('');
  const cancelForm = useForm<{ reason: string }>({
    resolver: zodResolver(reasonSchema),
    defaultValues: { reason: '' },
  });
  const reason = useWatch({ control: cancelForm.control, name: 'reason' });
  const setReason = (value: string) =>
    cancelForm.setValue('reason', value, { shouldDirty: true });
  const [reasonError, setReasonError] = useState('');
  const action = useMutation({
    mutationFn: async (task: {
      kind: 'approve' | 'reject' | 'remind' | 'cancel' | 'remove';
      participantId?: string;
      reason?: string;
    }) => {
      if (task.kind === 'remove') {
        await request({
          url: `/events/${eventId}/participants/${task.participantId}/remove`,
          method: 'POST',
          data: { reason: task.reason },
        });
        return 'Участник исключён.';
      }
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
    if (!(await cancelForm.trigger())) {
      setReasonError(
        cancelForm.formState.errors.reason?.message ?? 'Укажите причину отмены',
      );
      return;
    }
    setReasonError('');
    const accepted = await onConfirmCancel();
    if (accepted)
      action.mutate({
        kind: 'cancel',
        reason: cancelForm.getValues('reason'),
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
