import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  closeCompanyRequest,
  COMPANY_KEYS,
  createCompanyRequest,
} from '@/entities/company';

export const useCreateCompanyRequest = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: createCompanyRequest,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: COMPANY_KEYS.ALL });
    },
    meta: { silent: true },
  });
};
export const useCloseCompanyRequest = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: closeCompanyRequest,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: COMPANY_KEYS.ALL });
    },
    meta: { silent: true },
  });
};
