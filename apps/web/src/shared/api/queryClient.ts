import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query';

import { ApiError } from './errors';

const STALE_TIME = 30_000;

let onGlobalError: (error: ApiError) => void = (e) => console.error(e);

export const setErrorHandler = (fn: (error: ApiError) => void) => {
  onGlobalError = fn;
};

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => {
      if (query.meta?.silent) return;
      onGlobalError(error);
    },
  }),
  mutationCache: new MutationCache({
    onError: (error, _vars, _ctx, mutation) => {
      if (mutation.meta?.silent) return;
      onGlobalError(error);
    },
  }),
  defaultOptions: {
    queries: {
      staleTime: STALE_TIME,
      refetchOnWindowFocus: false,
      retry: (failureCount, error) => {
        if (error.status !== null && error.status < 500) return false;

        return failureCount < 2;
      },
    },
    mutations: { retry: false },
  },
});
