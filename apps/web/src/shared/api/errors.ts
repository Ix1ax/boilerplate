import axios from 'axios';

import { ErrorLogger } from '@/shared/lib';

type ApiErrorInit = {
  status?: number | null;
  data?: unknown;
  url?: string;
  method?: string;
  cause?: unknown;
  silent?: boolean;
};

type ApiErrorResponseBody = {
  error?: { message?: string };
  message?: string;
};

export class ApiError extends ErrorLogger {
  readonly status: number | null;
  readonly data: unknown;
  readonly url?: string;
  readonly method?: string;

  constructor(message: string, init: ApiErrorInit = {}) {
    const status = init.status ?? null;
    super(message, {
      scope: 'api',
      level: status !== null && status < 500 ? 'warn' : 'error',
      context: { status, method: init.method, url: init.url },
      cause: init.cause,
      silent: init.silent,
    });
    this.name = 'ApiError';
    this.status = status;
    this.data = init.data;
    this.url = init.url;
    this.method = init.method;
  }

  get isUnauthorized() {
    return this.status === 401;
  }
  get isForbidden() {
    return this.status === 403;
  }
  get isNetwork() {
    return this.status === null;
  }
}

export const toApiError = (error: unknown): ApiError => {
  if (error instanceof ApiError) return error;

  if (axios.isAxiosError(error)) {
    const status = error.response?.status ?? null;
    const data = error.response?.data as ApiErrorResponseBody | undefined;
    const message =
      data?.error?.message ??
      data?.message ??
      (status === null
        ? 'Нет соединения с сервером'
        : `Ошибка запроса (${status})`);
    return new ApiError(message, {
      status,
      data: error.response?.data,
      url: error.config?.url,
      method: error.config?.method?.toUpperCase(),
      cause: error,
    });
  }

  return new ApiError(
    error instanceof Error ? error.message : 'Неизвестная ошибка',
    {
      cause: error,
    },
  );
};
