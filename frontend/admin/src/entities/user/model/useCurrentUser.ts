import { useQuery } from '@tanstack/react-query';

import { getSession } from '@/shared/api';

export const useCurrentUser = () =>
  useQuery({
    queryKey: ['session'],
    queryFn: ({ signal }) => getSession(signal),
    retry: false,
    staleTime: 30_000,
  });
