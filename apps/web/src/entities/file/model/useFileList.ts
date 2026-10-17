import { useQuery } from '@tanstack/react-query';

import { FILE_KEYS } from './constants';
import type { FileMeta } from './types';

/**
 * У API нет эндпоинта для списка файлов (только POST/GET/DELETE по id) —
 * список ведём на клиенте: заполняется мутациями features/files через
 * queryClient.setQueryData по FILE_KEYS.LIST.
 */
export const useFileList = () => {
  return useQuery({
    queryKey: FILE_KEYS.LIST,
    queryFn: () => Promise.resolve<FileMeta[]>([]),
    initialData: [] as FileMeta[],
    staleTime: Infinity,
    gcTime: Infinity,
  });
};
