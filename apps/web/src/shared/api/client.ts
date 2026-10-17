import axios, {
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from 'axios';

import {
  API_BASE_URL,
  API_TIMEOUT,
  getApiConfig,
  SILENT_UNAUTHORIZED_URL,
} from './config';
import { toApiError } from './errors';
import { refreshSession, shouldSkipRefresh } from './refresh';

type RetriableRequest = InternalAxiosRequestConfig & { _retry?: boolean };

export const API = axios.create({
  baseURL: API_BASE_URL,
  timeout: API_TIMEOUT,
});

API.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (axios.isCancel(error)) return Promise.reject(error);

    const original = error.config as RetriableRequest | undefined;
    const canRefresh =
      error.response?.status === 401 &&
      !!original &&
      !original._retry &&
      !shouldSkipRefresh(original.url);

    if (canRefresh) {
      original._retry = true;
      try {
        await refreshSession();
        return API(original);
      } catch (refreshError) {
        const refreshApiError = toApiError(refreshError);
        // Молчим на /me: страница должна просто показать «не залогинен», а не
        // форсировать редирект — это решает ProtectedRoute/Header сами.
        if (
          (refreshApiError.isUnauthorized || refreshApiError.isForbidden) &&
          original.url !== SILENT_UNAUTHORIZED_URL
        ) {
          getApiConfig().onUnauthorized();
        }
        return Promise.reject(toApiError(error));
      }
    }

    const apiError = toApiError(error);
    if (apiError.isUnauthorized && original?.url !== SILENT_UNAUTHORIZED_URL)
      getApiConfig().onUnauthorized();
    return Promise.reject(apiError);
  },
);

export const request = <T>(config: AxiosRequestConfig) =>
  API.request<T>(config).then((response) => response.data);
