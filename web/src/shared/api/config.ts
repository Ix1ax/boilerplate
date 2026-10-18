import { env } from '@/shared/config';

export const API_BASE_URL = env.VITE_API_URL;
export const API_TIMEOUT = 15_000;

/**
 * Запрашивается на каждой странице (Header) даже без сессии — 401 здесь
 * означает «не залогинен», а не сбой, поэтому не должен форсировать редирект.
 */
export const SILENT_UNAUTHORIZED_URL = '/me';

export type ApiConfig = { onUnauthorized: () => void };

let current: ApiConfig = { onUnauthorized: () => {} };

export const configureApi = (next: ApiConfig) => {
  current = next;
};

export const getApiConfig = () => current;
