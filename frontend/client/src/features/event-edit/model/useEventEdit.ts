import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  type Event,
  eventPatchSchema,
  type EventPatchValues,
} from '@starter/contracts';
import { useQueryClient } from '@tanstack/react-query';

import { EVENT_KEYS, updateEvent } from '@/entities/event';

import { toApiError } from '@/shared/api';

const toFormValues = (event: Event): EventPatchValues => ({
  title: event.title,
  description: event.description,
  starts_at: event.starts_at,
  ends_at: event.ends_at ?? null,
  location: event.location,
  lat: event.lat ?? null,
  lng: event.lng ?? null,
  level: event.level,
  requirements: event.requirements,
  price_note: event.price_note,
  private_info: event.private_info,
  capacity: event.capacity ?? null,
  outdoor: event.outdoor,
});

export const useEventEdit = (event: Event) => {
  const cache = useQueryClient();
  const form = useForm<EventPatchValues>({
    resolver: zodResolver(eventPatchSchema),
    defaultValues: toFormValues(event),
  });
  const {
    handleSubmit,
    setError,
    reset,
    formState: { errors, isSubmitting, isSubmitSuccessful },
  } = form;

  // Карточка могла обновиться (другая вкладка, уведомление) — форма подхватывает актуальные значения.
  useEffect(() => {
    reset(toFormValues(event));
  }, [event, reset]);

  const submit = handleSubmit(async (values) => {
    try {
      const updated = await updateEvent(event.id, values);
      reset(toFormValues(updated));
      await cache.invalidateQueries({ queryKey: EVENT_KEYS.DETAIL(event.id) });
    } catch (error) {
      setError('root', { message: toApiError(error).message });
    }
  });

  return { form, submit, errors, isSubmitting, isSubmitSuccessful };
};
