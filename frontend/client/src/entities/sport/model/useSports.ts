import { useQuery } from '@tanstack/react-query';

import { getSports } from '../api';
import { SPORT_KEYS } from './constants';

// Справочник меняется редко: свои направления появляются после создания встречи.
export const useSports = (options: { enabled?: boolean } = {}) =>
  useQuery({
    queryKey: SPORT_KEYS.LIST,
    queryFn: getSports,
    staleTime: 5 * 60_000,
    enabled: options.enabled ?? true,
  });
