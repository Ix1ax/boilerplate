import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { eventEditSchema, type EventPatchValues } from '@starter/contracts';
import { useQueryClient } from '@tanstack/react-query';

import { type Event, EVENT_KEYS, updateEvent } from '@/entities/event';

import { toApiError } from '@/shared/api';

const toFormValues = (event: Event): EventPatchValues => ({
  zone: event.zone,
  meeting_host: event.meetingHost,
  step_free: event.stepFree,
  companion_allowed: event.companionAllowed,
  title: event.title,
  description: event.description,
  starts_at: event.startsAt,
  ends_at: event.endsAt ?? null,
  location: event.location,
  lat: event.lat ?? null,
  lng: event.lng ?? null,
  level: event.level,
  requirements: event.requirements,
  price_note: event.priceNote,
  private_info: event.privateInfo,
  capacity: event.capacity ?? null,
  outdoor: event.outdoor,
});

export const useEventEdit = (event: Event) => {
  const cache = useQueryClient();
  const form = useForm<EventPatchValues>({
    resolver: zodResolver(eventEditSchema),
    defaultValues: toFormValues(event),
  });
  const {
    handleSubmit,
    setError,
    reset,
    formState: { errors, isSubmitting, isSubmitSuccessful, dirtyFields },
  } = form;

  // Карточка могла обновиться (другая вкладка, уведомление) — форма подхватывает актуальные значения.
  useEffect(() => {
    reset(toFormValues(event), { keepDirtyValues: true });
  }, [event, reset]);

  const submit = handleSubmit(async (values) => {
    try {
      const patch: EventPatchValues = Object.fromEntries(
        Object.entries(values).filter(([key]) => key in dirtyFields),
      );
      // Координаты изменяются только парой.
      if ('lat' in patch || 'lng' in patch) {
        patch.lat = values.lat ?? null;
        patch.lng = values.lng ?? null;
      }
      const updated = await updateEvent(event.id, patch);
      reset(toFormValues(updated));
      await cache.invalidateQueries({ queryKey: EVENT_KEYS.ALL });
    } catch (error) {
      setError('root', { message: toApiError(error).message });
    }
  });

  return { form, submit, errors, isSubmitting, isSubmitSuccessful };
};
