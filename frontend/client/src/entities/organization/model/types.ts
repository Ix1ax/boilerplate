import type { Event, Organization, Profile } from '@starter/contracts';

export type { Organization, OrganizationValues } from '@starter/contracts';

export type PublicOrganization = {
  organization: Organization;
  stats: Profile['stats'];
  events: Event[];
};
