import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useSearchParams } from 'react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  type EventFormInput,
  type EventFormValues,
  eventInputSchema,
} from '@starter/contracts';
import { useQueryClient } from '@tanstack/react-query';

import { createEvent, type Event, EVENT_KEYS } from '@/entities/event';
import { useMyOrganization } from '@/entities/organization';
import { SPORT_KEYS, useSports } from '@/entities/sport';

import { toApiError } from '@/shared/api';

export const useEventCreate = () => {
  const [search] = useSearchParams();
  const [created, setCreated] = useState<Event | null>(null);
  const cache = useQueryClient();
  const sportsQuery = useSports();
  const sports = sportsQuery.data ?? [];
  const organization = useMyOrganization();
  const form = useForm<EventFormInput, unknown, EventFormValues>({
    resolver: zodResolver(eventInputSchema),
    defaultValues: {
      title: '',
      description: '',
      location: '',
      level: (['any', 'beginner', 'amateur', 'pro'].find(
        (v) => v === search.get('level'),
      ) ?? 'any') as EventFormInput['level'],
      requirements: '',
      price_note: '',
      private_info: '',
      join_mode: 'open',
      as_organization: search.get('as_organization') === '1',
      outdoor: false,
      sport_id: search.get('sport_id') || undefined,
      sport_name: search.get('sport_name') || undefined,
      starts_at: search.get('starts_at') || undefined,
      ends_at: search.get('ends_at') || undefined,
      zone: search.get('zone') || '',
      meeting_host: '',
      step_free: false,
      companion_allowed: false,
    },
  });
  const {
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = form;
  const submit = handleSubmit(async (values) => {
    try {
      const event = await createEvent(values);
      await Promise.all([
        cache.invalidateQueries({ queryKey: EVENT_KEYS.ALL }),
        cache.invalidateQueries({ queryKey: SPORT_KEYS.LIST }),
      ]);
      setCreated(event);
    } catch (error) {
      setError('root', { message: toApiError(error).message });
    }
  });
  return {
    form,
    created,
    submit,
    errors,
    isSubmitting,
    sports,
    sportsQuery,
    organization,
  };
};
