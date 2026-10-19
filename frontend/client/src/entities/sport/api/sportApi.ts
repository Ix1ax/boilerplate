import { request } from '@/shared/api';

import type { Demand, Sport } from '../model/types';

export const getSports = () =>
  request<{ items: Sport[] }>({ url: '/sports' }).then((data) => data.items);

export type DemandParams = {
  sportId?: string;
  sportName?: string;
  level?: string;
};

export const getDemand = ({ sportId, sportName, level }: DemandParams) =>
  request<Demand>({
    url: '/sports/demand',
    params: { sport_id: sportId, sport_name: sportName, level },
  });
