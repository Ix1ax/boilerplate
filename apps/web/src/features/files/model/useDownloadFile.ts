import { useMutation } from '@tanstack/react-query';

import { downloadFile } from '../api';

const saveBlob = (blob: Blob, name: string) => {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
};

export const useDownloadFile = () => {
  return useMutation({
    mutationFn: async ({ id, name }: { id: string; name: string }) => {
      const blob = await downloadFile(id);
      saveBlob(blob, name);
    },
  });
};
