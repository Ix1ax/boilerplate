import axios, { type AxiosRequestConfig } from 'axios';

import { API_BASE_URL, API_TIMEOUT, getApiConfig } from './config';
import { toApiError } from './errors';
export const API = axios.create({
  baseURL: API_BASE_URL,
  timeout: API_TIMEOUT,
});
API.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (axios.isCancel(error)) return Promise.reject(error);
    const apiError = toApiError(error);
    if (apiError.isUnauthorized && error.config?.url !== '/me')
      getApiConfig().onUnauthorized();
    return Promise.reject(apiError);
  },
);
export const request = <T>(config: AxiosRequestConfig) =>
  API.request<T>(config).then((response) => response.data);
