import axios, {
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from 'axios';

import { API_BASE_URL, API_TIMEOUT, getApiConfig } from './config';
import { toApiError } from './errors';
import { refreshAccessToken, shouldSkipRefresh } from './refresh';

type RetriableRequest = InternalAxiosRequestConfig & { _retry?: boolean };

export const API = axios.create({
  baseURL: API_BASE_URL,
  timeout: API_TIMEOUT,
  headers: {
    'Content-Type': 'application/json',
  },
});

API.interceptors.request.use((req) => {
  const token = getApiConfig().getAccessToken();
  if (token) req.headers.Authorization = `Bearer ${token}`;
  return req;
});

API.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (axios.isCancel(error)) return Promise.reject(error);

    const { getRefreshToken, onUnauthorized } = getApiConfig();

    const original = error.config as RetriableRequest | undefined;
    const canRefresh =
      error.response?.status === 401 &&
      !!original &&
      !original._retry &&
      !shouldSkipRefresh(original.url) &&
      !!getRefreshToken();

    if (canRefresh) {
      original._retry = true;

      try {
        await refreshAccessToken();
        return API(original);
      } catch (refreshError) {
        const apiError = toApiError(refreshError);
        if (apiError.isUnauthorized || apiError.isForbidden) onUnauthorized();
        return Promise.reject(apiError);
      }
    }

    const apiError = toApiError(error);
    if (apiError.isUnauthorized) onUnauthorized();

    return Promise.reject(apiError);
  },
);

export const request = <T>(cfg: AxiosRequestConfig) =>
  API.request<T>(cfg).then((r) => r.data);
