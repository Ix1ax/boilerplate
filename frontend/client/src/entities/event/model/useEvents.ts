import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { getEvents } from '../api';
import { EVENT_KEYS } from './constants';
import type { EventListParams } from './types';

export const useEvents = (
  params: EventListParams = {},
  options: { enabled?: boolean } = {},
) =>
  useQuery({
    queryKey: EVENT_KEYS.LIST(params),
    queryFn: () => getEvents(params),
    // Фильтры меняются часто: старый список остаётся на экране, пока грузится новый.
    placeholderData: keepPreviousData,
    enabled: options.enabled ?? true,
    meta: { silent: true },
  });
