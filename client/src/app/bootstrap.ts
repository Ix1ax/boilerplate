import { toast } from 'sonner';

import { configureApi, setErrorHandler } from '@/shared/api';
import { env } from '@/shared/config';
import { STATIC_ROUTES } from '@/shared/constants';
import { initGlobalErrorHandlers, logger } from '@/shared/lib';
import { configureWs } from '@/shared/ws';

export const bootstrap = () => {
  initGlobalErrorHandlers();

  try {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
  } catch (error) {
    logger
      .child('bootstrap')
      .warn('Не удалось очистить localStorage от старых токенов', error);
  }
  // Сессия истекла посреди работы: после входа вернём на ту же страницу.
  const toLogin = () =>
    window.location.assign(
      `${STATIC_ROUTES.LOGIN}?next=${encodeURIComponent(window.location.pathname + window.location.search)}`,
    );
  configureApi({ onUnauthorized: toLogin });
  setErrorHandler((error) => toast.error(error.message));

  configureWs({
    getUrl: () =>
      env.VITE_WS_URL ||
      `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}${env.VITE_API_URL}/ws`,
    getToken: () => null,
    onUnauthorized: toLogin,
  });
};
