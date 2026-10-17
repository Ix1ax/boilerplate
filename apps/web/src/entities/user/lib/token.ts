import { invalidateSession, type Tokens } from '@/shared/api';
import { STORAGE_KEYS } from '@/shared/constants';
import {
  readLocalStorage,
  removeLocalStorage,
  writeLocalStorage,
} from '@/shared/lib';

export const getAccessToken = () => readLocalStorage(STORAGE_KEYS.ACCESS_TOKEN);
export const getRefreshToken = () =>
  readLocalStorage(STORAGE_KEYS.REFRESH_TOKEN);
export const hasSession = () => !!getAccessToken() || !!getRefreshToken();

export const setTokens = ({ accessToken, refreshToken }: Tokens) => {
  // Новый набор токенов = новая сессия: гасим in-flight refresh прошлой
  // (важно при смене аккаунта без явного logout).
  invalidateSession();
  writeLocalStorage(STORAGE_KEYS.ACCESS_TOKEN, accessToken);
  if (refreshToken) writeLocalStorage(STORAGE_KEYS.REFRESH_TOKEN, refreshToken);
};

export const clearTokens = () => {
  invalidateSession();
  removeLocalStorage(STORAGE_KEYS.ACCESS_TOKEN);
  removeLocalStorage(STORAGE_KEYS.REFRESH_TOKEN);
};
