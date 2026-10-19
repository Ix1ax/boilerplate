import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  ORGANIZATION_KEYS,
  type OrganizationValues,
  submitOrganization,
  updateMyOrganization,
} from '@/entities/organization';

export const useSaveOrganization = (existing: boolean) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (data: OrganizationValues) =>
      existing ? updateMyOrganization(data) : submitOrganization(data),
    onSuccess: (data) => {
      client.setQueryData(ORGANIZATION_KEYS.MY, data);
    },
    meta: { silent: true },
  });
};
