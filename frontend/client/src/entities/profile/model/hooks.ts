import { useQuery } from '@tanstack/react-query';

import { getProfile } from '../api';

export const usePersonProfile = (id: string) => {
  return useQuery({
    queryKey: ['profile', id],
    queryFn: () => getProfile(id),
    enabled: Boolean(id),
    meta: { silent: true },
  });
};
