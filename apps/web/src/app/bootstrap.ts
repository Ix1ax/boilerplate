import { toast } from 'sonner';

import { configureApi, setErrorHandler } from '@/shared/api';
import { STATIC_ROUTES } from '@/shared/constants';
import { initGlobalErrorHandlers } from '@/shared/lib';
export const bootstrap = () => {
  initGlobalErrorHandlers();
  // Удаляем токены, которые мог сохранить предыдущий вариант фронтенда.
  try {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
  } catch {
    /* Хранилище может быть запрещено браузером. */
  }
  configureApi({
    onUnauthorized: () => window.location.assign(STATIC_ROUTES.LOGIN),
  });
  setErrorHandler((error) => toast.error(error.message));
};
