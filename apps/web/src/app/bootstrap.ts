import { toast } from 'sonner';

import { configureApi, setErrorHandler } from '@/shared/api';
import { STATIC_ROUTES } from '@/shared/constants';
import { initGlobalErrorHandlers, logger } from '@/shared/lib';

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
};
