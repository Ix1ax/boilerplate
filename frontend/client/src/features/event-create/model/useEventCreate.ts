import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  type EventFormInput,
  type EventFormValues,
  eventInputSchema,
} from '@starter/contracts';
import { useQueryClient } from '@tanstack/react-query';

import { createEvent, EVENT_KEYS } from '@/entities/event';
import { useMyOrganization } from '@/entities/organization';
import { SPORT_KEYS, useSports } from '@/entities/sport';

import { toApiError } from '@/shared/api';
import { DYNAMIC_ROUTES } from '@/shared/constants';

export const useEventCreate = () => {
  const navigate = useNavigate();
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
      level: 'any',
      requirements: '',
      price_note: '',
      private_info: '',
      join_mode: 'open',
      as_organization: false,
      outdoor: false,
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
      void navigate(DYNAMIC_ROUTES.EVENT(event.id));
    } catch (error) {
      setError('root', { message: toApiError(error).message });
    }
  });
  return {
    form,
    submit,
    errors,
    isSubmitting,
    sports,
    sportsQuery,
    organization,
  };
};
