import type { FilePage } from '@starter/contracts';
import { useQuery } from '@tanstack/react-query';

import { request } from '@/shared/api';

import { FILE_KEYS } from './constants';
import { toFileMeta } from './mappers';

export const useFileList = (offset = 0, limit = 20) =>
  useQuery({
    queryKey: [...FILE_KEYS.LIST, { offset, limit }],
    queryFn: () =>
      request<FilePage>({ url: '/files', params: { offset, limit } }).then(
        (page) => ({ ...page, items: page.items.map(toFileMeta) }),
      ),
  });
