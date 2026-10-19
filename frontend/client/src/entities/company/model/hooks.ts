import { useQuery } from '@tanstack/react-query';

import { listCompanyMatches, listMyCompanyRequests } from '../api';
import { COMPANY_KEYS } from './constants';

export const useCompanyRequests = () => {
  return useQuery({
    queryKey: COMPANY_KEYS.MY,
    queryFn: listMyCompanyRequests,
    meta: { silent: true },
  });
};

export const useCompanyMatches = (id: string, offset = 0) => {
  return useQuery({
    queryKey: COMPANY_KEYS.MATCHES(id, offset),
    queryFn: () => listCompanyMatches(id, offset),
    meta: { silent: true },
  });
};
