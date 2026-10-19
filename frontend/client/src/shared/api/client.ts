import axios, { type AxiosRequestConfig } from 'axios';

import { deepCamelKeys } from './caseTransform';
import {
  API_BASE_URL,
  API_TIMEOUT,
  getApiConfig,
  SILENT_UNAUTHORIZED_URL,
} from './config';
import { toApiError } from './errors';

// Same-origin cookie браузер отправляет сам. Токены хранятся и обновляются только в Go.
export const API = axios.create({
  baseURL: API_BASE_URL,
  timeout: API_TIMEOUT,
});

// Go отдаёт JSON в snake_case — переводим ключи в camelCase один раз, на границе API.
API.interceptors.response.use((response) => {
  response.data = deepCamelKeys(response.data);
  return response;
});

API.interceptors.response.use(
  (response) => response,
  (error) => {
    if (axios.isCancel(error)) return Promise.reject(error);

    // 401 на /me допустим для гостя. Ошибки сети и 503 не должны отправлять пользователя на вход.
    const apiError = toApiError(error);
    if (
      apiError.isUnauthorized &&
      error.config?.url !== SILENT_UNAUTHORIZED_URL &&
      !['/auth/local/login', '/auth/local/register'].includes(
        error.config?.url ?? '',
      )
    )
      getApiConfig().onUnauthorized();
    return Promise.reject(apiError);
  },
);

export const request = <T>(config: AxiosRequestConfig) =>
  API.request<T>(config).then((response) => response.data);
