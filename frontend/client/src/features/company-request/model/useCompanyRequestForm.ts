import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { companyRequestSchema } from '@starter/contracts';
import type { z } from 'zod';

import {
  type CompanyRequest,
  type CompanyRequestValues,
} from '@/entities/company';

import { useCreateCompanyRequest } from './useCompanyRequestActions';

export const useCompanyRequestForm = (
  onCreated: (request: CompanyRequest) => void,
) => {
  const form = useForm<
    z.input<typeof companyRequestSchema>,
    unknown,
    CompanyRequestValues
  >({
    resolver: zodResolver(companyRequestSchema),
    defaultValues: {
      sport_name: '',
      level: 'any',
      area: '',
      note: '',
      weekdays: [],
      day_parts: [],
    },
  });
  const create = useCreateCompanyRequest();
  const submit = form.handleSubmit((data) =>
    create.mutate(data, {
      onSuccess: (request) => {
        form.reset();
        onCreated(request);
      },
    }),
  );
  return { form, create, submit };
};
