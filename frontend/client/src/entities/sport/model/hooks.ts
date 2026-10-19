import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { type DemandParams, getDemand, getSports } from '../api';
import { SPORT_KEYS } from './constants';

export const useDemand = (params: DemandParams) =>
  useQuery({
    queryKey: SPORT_KEYS.DEMAND(params),
    queryFn: () => getDemand(params),
    enabled: !!(params.sportId || params.sportName?.trim()),
    placeholderData: keepPreviousData,
    meta: { silent: true },
  });

// Справочник меняется редко: свои направления появляются после создания встречи.
export const useSports = (options: { enabled?: boolean } = {}) =>
  useQuery({
    queryKey: SPORT_KEYS.LIST,
    enabled: options.enabled ?? true,
    queryFn: getSports,
    staleTime: 5 * 60_000,
  });
