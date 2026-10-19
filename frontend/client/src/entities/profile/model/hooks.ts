import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { getMyProfile, getProfile, updateMyProfile } from '../api';

export const usePersonProfile = (id: string) => {
  return useQuery({
    queryKey: ['profile', id],
    queryFn: () => getProfile(id),
    enabled: Boolean(id),
    meta: { silent: true },
  });
};

export const useMyProfile = (enabled = true) =>
  useQuery({
    queryKey: ['profile', 'me'],
    queryFn: getMyProfile,
    enabled,
    meta: { silent: true },
  });
export const useUpdateMyProfile = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: updateMyProfile,
    meta: { silent: true },
    onSuccess: (data) => {
      client.setQueryData(['profile', 'me'], data);
      void client.invalidateQueries({ queryKey: ['profile', data.id] });
    },
  });
};
