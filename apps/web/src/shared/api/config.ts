import { env } from '@/shared/config';
export const API_BASE_URL = env.VITE_API_URL;
export const API_TIMEOUT = 15_000;
export type ApiConfig = { onUnauthorized: () => void };
let current: ApiConfig = { onUnauthorized: () => {} };
export const configureApi = (next: ApiConfig) => {
  current = next;
};
export const getApiConfig = () => current;
