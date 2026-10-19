import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { type DemandParams, getDemand } from '../api';
import { SPORT_KEYS } from './constants';

export const useDemand = (params: DemandParams) =>
  useQuery({
    queryKey: SPORT_KEYS.DEMAND(params),
    queryFn: () => getDemand(params),
    enabled: !!(params.sportId || params.sportName?.trim()),
    placeholderData: keepPreviousData,
    meta: { silent: true },
  });
