import type { CompanyPlan as CompanyRequestDto } from '@starter/contracts';

import type { EventPage } from '@/entities/event';

import type { Camelize } from '@/shared/api';

export type { CompanyPlanValues as CompanyRequestValues } from '@starter/contracts';

export type CompanyRequest = Camelize<CompanyRequestDto>;
export type SportEventPage = EventPage;
