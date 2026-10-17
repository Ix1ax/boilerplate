import { env } from '@/shared/config';

export const API_BASE_URL = env.VITE_API_URL;
export const API_TIMEOUT = 15_000;

export type Tokens = {
  accessToken: string;
  refreshToken?: string;
};

export type ApiConfig = {
  getAccessToken: () => string | null;
  getRefreshToken: () => string | null;
  saveTokens: (tokens: Tokens) => void;
  onUnauthorized: () => void;
};

let current: ApiConfig = {
  getAccessToken: () => null,
  getRefreshToken: () => null,
  saveTokens: () => {},
  onUnauthorized: () => {},
};

export const configureApi = (next: ApiConfig) => {
  current = next;
};

export const getApiConfig = () => current;
