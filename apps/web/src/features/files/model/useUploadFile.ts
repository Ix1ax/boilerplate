import { useMutation, useQueryClient } from '@tanstack/react-query';

import { FILE_KEYS, type FileMeta } from '@/entities/file';

import { uploadFile } from '../api';

export const useUploadFile = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: uploadFile,
    onSuccess: (file) => {
      queryClient.setQueryData<FileMeta[]>(FILE_KEYS.LIST, (files = []) => [
        file,
        ...files,
      ]);
    },
  });
};
