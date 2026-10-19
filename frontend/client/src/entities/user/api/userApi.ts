import { type ApiError, request, SILENT_UNAUTHORIZED_URL } from '@/shared/api';

import type { User } from '../model';

export const getMe = () =>
  request<User | null>({ url: SILENT_UNAUTHORIZED_URL }).catch(
    (error: ApiError) => {
      // Только 401 означает гостя; сбой API должен остаться ошибкой с возможностью повтора.
      if (error.isUnauthorized) return null;
      throw error;
    },
  );
