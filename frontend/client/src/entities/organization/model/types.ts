import type { Event, Organization, Profile } from '@starter/contracts';

export type { Organization, OrganizationValues } from '@starter/contracts';
export type { OrganizationDetail, OrganizationPage } from '@starter/contracts';

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
