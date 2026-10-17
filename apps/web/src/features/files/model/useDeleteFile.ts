import { useMutation, useQueryClient } from '@tanstack/react-query';

import { FILE_KEYS } from '@/entities/file';

import { deleteFile } from '../api';
export const useDeleteFile = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deleteFile,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: FILE_KEYS.ALL }),
  });
};
