import { toast } from 'sonner';

import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  setTokens,
} from '@/entities/user';

import { configureApi, setErrorHandler } from '@/shared/api';
import { env } from '@/shared/config';
import { STATIC_ROUTES } from '@/shared/constants';
import { initGlobalErrorHandlers } from '@/shared/lib';
import { configureWs } from '@/shared/ws';

const onUnauthorized = () => {
  clearTokens();
  if (window.location.pathname !== STATIC_ROUTES.LOGIN) {
    window.location.assign(STATIC_ROUTES.LOGIN);
  }
};

export const bootstrap = () => {
  initGlobalErrorHandlers();

  configureApi({
    getAccessToken,
    getRefreshToken,
    saveTokens: setTokens,
    onUnauthorized,
  });

  setErrorHandler((error) => {
    toast.error(error.message);
  });

  configureWs({
    getUrl: () => env.VITE_WS_URL,
    getToken: getAccessToken,
    onUnauthorized,
  });
};
