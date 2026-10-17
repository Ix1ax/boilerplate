import { API } from './client';

const REFRESH_URL = '/auth/refresh';

/** Для этих URL 401 не повод продлевать сессию — только обычная обработка. */
export const shouldSkipRefresh = (url?: string) =>
  !url || url === REFRESH_URL || url === '/auth/logout';

let inflight: Promise<unknown> | null = null;

/** Продлевает сессию (sliding window) на бэке; параллельные запросы делят один вызов. */
export const refreshSession = () => {
  inflight ??= API.post(REFRESH_URL, undefined, { baseURL: '/' }).finally(
    () => {
      inflight = null;
    },
  );
  return inflight;
};
