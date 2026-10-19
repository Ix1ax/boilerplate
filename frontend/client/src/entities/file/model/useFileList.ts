import type { FilePage } from '@starter/contracts';
import { useQuery } from '@tanstack/react-query';

import { request } from '@/shared/api';

import { FILE_KEYS } from './constants';
import { toFileMeta } from './mappers';
import type { FileEntityRef } from './types';

type FileListParams = {
  offset?: number;
  limit?: number;
  entity?: FileEntityRef;
  enabled?: boolean;
};

export const useFileList = ({
  offset = 0,
  limit = 20,
  entity,
  enabled = true,
}: FileListParams = {}) =>
  useQuery({
    queryKey: [...FILE_KEYS.LIST, { offset, limit, entity }],
    enabled,
    queryFn: () =>
      request<FilePage>({
        url: '/files',
        params: {
          offset,
          limit,
          entity_type: entity?.type,
          entity_id: entity?.id,
        },
      }).then((page) => ({ ...page, items: page.items.map(toFileMeta) })),
  });
