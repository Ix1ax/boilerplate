import type {
  Organization as OrganizationDto,
  OrganizationDetail as OrganizationDetailDto,
  OrganizationPage as OrganizationPageDto,
} from '@starter/contracts';

import type { Event } from '@/entities/event';
import type { Profile } from '@/entities/profile';

import type { Camelize } from '@/shared/api';

export type { OrganizationProfileValues as OrganizationValues } from '@starter/contracts';

export type Organization = Camelize<OrganizationDto> & {
  site: string;
  zones: string[];
  venues: string[];
};
export type OrganizationDetail = Camelize<OrganizationDetailDto>;
export type OrganizationPage = Camelize<OrganizationPageDto>;

export type PublicOrganization = {
  organization: Organization;
  stats: Profile['stats'];
  events: Event[];
};

export type AdminOrganizationsParams = {
  limit?: number;
  offset?: number;
  status?: string;
  q?: string;
};
