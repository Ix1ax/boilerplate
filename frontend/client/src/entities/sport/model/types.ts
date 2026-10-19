import type { Demand as DemandDto, Sport as SportDto } from '@starter/contracts';

import type { Camelize } from '@/shared/api';

export type Sport = Camelize<SportDto>;
export type Demand = Camelize<DemandDto>;
