import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  approveOrganization,
  ORGANIZATION_KEYS,
  rejectOrganization,
} from '@/entities/organization';

export const useApproveOrganization = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => approveOrganization(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ORGANIZATION_KEYS.ALL });
    },
  });
};

export const useRejectOrganization = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      rejectOrganization(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ORGANIZATION_KEYS.ALL });
    },
  });
};
