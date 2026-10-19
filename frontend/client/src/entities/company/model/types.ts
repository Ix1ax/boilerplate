import type { CompanyRequest as CompanyRequestDto } from '@starter/contracts';

import type { EventPage } from '@/entities/event';

import type { Camelize } from '@/shared/api';

export type { CompanyRequestValues } from '@starter/contracts';

export type CompanyRequest = Camelize<CompanyRequestDto>;
export type SportEventPage = EventPage;
