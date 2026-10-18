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
  configureApi({
    onUnauthorized: () => window.location.assign(STATIC_ROUTES.LOGIN),
  });
  setErrorHandler((error) => toast.error(error.message));
  // WebSocket идёт через gateway на том же origin: сессия — HttpOnly cookie, токен в URL не нужен.
  configureWs({
    getUrl: () =>
      env.VITE_WS_URL ||
      `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}${env.VITE_API_URL}/ws`,
    getToken: () => null,
    onUnauthorized: () => window.location.assign(STATIC_ROUTES.LOGIN),
  });
};
