import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { organizationApplicationSchema } from '@starter/contracts';
import type { z } from 'zod';

import { type Organization } from '@/entities/organization';

import { useSaveOrganization } from './useSaveOrganization';

export const useClubForm = (organization: Organization | null) => {
  const form = useForm<
    z.input<typeof organizationApplicationSchema>,
    unknown,
    z.output<typeof organizationApplicationSchema>
  >({
    resolver: zodResolver(organizationApplicationSchema),
    defaultValues: {
      site: '',
      zones: [],
      venues: [],
      representative: Boolean(organization),
      name: '',
      description: '',
      city: '',
      contacts: '',
      sport_ids: [],
    },
  });
  useEffect(() => {
    if (organization && !form.formState.isDirty)
      form.reset({
        site: organization.site ?? '',
        zones: organization.zones ?? [],
        venues: organization.venues ?? [],
        representative: true,
        name: organization.name,
        description: organization.description,
        city: organization.city,
        contacts: organization.contacts,
        sport_ids: organization.sports.map((s) => s.id),
      });
  }, [organization, form]);
  const save = useSaveOrganization(Boolean(organization));
  const submit = form.handleSubmit((data) =>
    save.mutate({
      name: data.name,
      description: data.description,
      city: data.city,
      contacts: data.contacts,
      sport_ids: data.sport_ids,
      site: data.site,
      zones: data.zones,
      venues: data.venues,
    }),
  );
  return { form, save, submit };
};
