import { useMutation, useQueryClient } from '@tanstack/react-query';

import { FILE_KEYS } from '@/entities/file';

import { uploadFile } from '../api';
export const useUploadFile = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: uploadFile,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: FILE_KEYS.ALL }),
  });
};
