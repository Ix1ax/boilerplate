import type { ApiError } from '@/shared/api';

declare module '@tanstack/react-query' {
  interface Register {
    defaultError: ApiError;
    queryMeta: { silent?: boolean };
    mutationMeta: { silent?: boolean };
  }
}
