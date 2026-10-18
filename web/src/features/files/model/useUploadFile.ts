import { useMutation, useQueryClient } from '@tanstack/react-query';

import { FILE_KEYS, type FileEntityRef } from '@/entities/file';

import { uploadFile } from '../api';

export const useUploadFile = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ file, entity }: { file: File; entity?: FileEntityRef }) =>
      uploadFile(file, entity),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: FILE_KEYS.ALL }),
  });
};
