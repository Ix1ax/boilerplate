import axios from 'axios';

import { API_BASE_URL, API_TIMEOUT, getApiConfig, type Tokens } from './config';
import { ApiError } from './errors';

export const LOGIN_URL = '/auth/login';
export const REFRESH_URL = '/auth/refresh';

const NO_REFRESH_URLS = [LOGIN_URL, REFRESH_URL];

export const shouldSkipRefresh = (url?: string) => {
  if (!url) return false;
  const path = url.split(/[?#]/, 1)[0];

  return NO_REFRESH_URLS.some((u) => path === u || path.endsWith(u));
};

// Версия (эпоха) сессии. Логаут/смена аккаунта инкрементят её через
// invalidateSession(); refresh применяет результат только к своей эпохе.
let sessionEpoch = 0;

/**
 * Завершить текущую сессию: запоздалый ответ уже запущенного refresh не вернёт
 * токены в хранилище. Вызывается при очистке сессии (logout/onUnauthorized).
 */
export const invalidateSession = () => {
  sessionEpoch += 1;
  pending = null;
};

const requestTokens = async () => {
  const epoch = sessionEpoch;
  const { getRefreshToken, saveTokens } = getApiConfig();
  const { data } = await axios.post<Tokens>(
    REFRESH_URL,
    { refreshToken: getRefreshToken() },
    { baseURL: API_BASE_URL, timeout: API_TIMEOUT },
  );
  // Сессия сменилась/завершилась, пока шёл refresh — не трогаем хранилище.
  // Это штатная отмена, а не ошибка: silent, чтобы ErrorLogger не шумел в консоль.
  if (epoch !== sessionEpoch) {
    throw new ApiError('Refresh отменён: сессия завершена', { silent: true });
  }
  saveTokens(data);

  return data.accessToken;
};

let pending: Promise<string> | null = null;

export const refreshAccessToken = () => {
  pending ??= requestTokens().finally(() => {
    pending = null;
  });
  return pending;
};
