import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  type CompanyPlanInput,
  companyPlanSchema,
  type CompanyPlanValues,
} from '@starter/contracts';
import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  COMPANY_KEYS,
  type CompanyRequest,
  createCompanyRequest,
  updateCompanyRequest,
} from '@/entities/company';

export const useCompanyRequestForm = (
  onCreated: (request: CompanyRequest) => void,
  editing?: CompanyRequest,
) => {
  const [initialExpiry] = useState(() =>
    new Date(Date.now() + 7 * 86_400_000).toISOString(),
  );
  const form = useForm<CompanyPlanInput, unknown, CompanyPlanValues>({
    resolver: zodResolver(companyPlanSchema),
    defaultValues: {
      sport_name: '',
      level: 'any',
      area: '',
      note: '',
      weekdays: [],
      day_parts: [],
      zones: [],
      windows: [],
      expires_at: initialExpiry,
    },
  });
  const { reset } = form;
  useEffect(() => {
    reset(
      editing
        ? {
            sport_id: editing.sport.id,
            sport_name: editing.sport.name,
            level: editing.level,
            area: editing.area,
            note: editing.note,
            weekdays: [],
            day_parts: [],
            zones: editing.zones,
            windows: editing.windows.map((w) => ({
              starts_at: w.startsAt,
              ends_at: w.endsAt,
            })),
            expires_at: editing.expiresAt,
          }
        : {
            sport_name: '',
            level: 'any',
            area: '',
            note: '',
            weekdays: [],
            day_parts: [],
            zones: [],
            windows: [],
            expires_at: initialExpiry,
          },
    );
  }, [editing, reset, initialExpiry]);
  const cache = useQueryClient();
  const create = useMutation({
    mutationFn: (data: CompanyPlanValues) =>
      editing
        ? updateCompanyRequest(editing.id, data)
        : createCompanyRequest(data),
    onSuccess: (item) => {
      void cache.invalidateQueries({ queryKey: COMPANY_KEYS.ALL });
      if (!editing) form.reset();
      onCreated(item);
    },
    meta: { silent: true },
  });
  const submit = form.handleSubmit((data) => create.mutate(data));
  return { form, create, submit };
};
