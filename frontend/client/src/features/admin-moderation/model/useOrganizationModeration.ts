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
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin'] });
      void queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: ORGANIZATION_KEYS.ALL });
    },
  });
};

export const useRejectOrganization = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      rejectOrganization(id, reason),
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin'] });
      void queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: ORGANIZATION_KEYS.ALL });
    },
  });
};
