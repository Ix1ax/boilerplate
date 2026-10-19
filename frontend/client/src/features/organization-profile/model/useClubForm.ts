import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { organizationInputSchema } from '@starter/contracts';
import type { z } from 'zod';

import {
  type Organization,
  type OrganizationValues,
} from '@/entities/organization';

import { useSaveOrganization } from './useSaveOrganization';

export const useClubForm = (organization: Organization | null) => {
  const form = useForm<
    z.input<typeof organizationInputSchema>,
    unknown,
    OrganizationValues
  >({
    resolver: zodResolver(organizationInputSchema),
    defaultValues: {
      name: '',
      description: '',
      city: '',
      contacts: '',
      sport_ids: [],
    },
  });
  useEffect(() => {
    if (organization)
      form.reset({
        name: organization.name,
        description: organization.description,
        city: organization.city,
        contacts: organization.contacts,
        sport_ids: organization.sports.map((s) => s.id),
      });
  }, [organization, form]);
  const save = useSaveOrganization(Boolean(organization));
  const submit = form.handleSubmit((data) => save.mutate(data));
  return { form, save, submit };
};
