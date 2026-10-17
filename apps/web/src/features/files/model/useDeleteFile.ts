import { useMutation, useQueryClient } from '@tanstack/react-query';

import { FILE_KEYS, type FileMeta } from '@/entities/file';

import { deleteFile } from '../api';

export const useDeleteFile = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteFile,
    onSuccess: (_, id) => {
      queryClient.setQueryData<FileMeta[]>(FILE_KEYS.LIST, (files = []) =>
        files.filter((file) => file.id !== id),
      );
    },
  });
};
