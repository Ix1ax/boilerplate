import { Navigate, Outlet, useLocation } from 'react-router';

import { hasRole, useCurrentUser, type UserRole } from '@/entities/user';

import { STATIC_ROUTES } from '@/shared/constants';
import { ErrorScreen, LoaderScreen } from '@/shared/ui';

interface ProtectedRouteProps {
  roles?: UserRole[];
}

export const ProtectedRoute = ({ roles }: ProtectedRouteProps) => {
  const { data: user, isLoading, isError, refetch } = useCurrentUser();
  const location = useLocation();

  if (isLoading) return <LoaderScreen />;

  // При сбое проверки предлагаем повторить запрос, а не считаем пользователя вышедшим.
  if (isError && !user) {
    return (
      <ErrorScreen
        description="Не удалось проверить сессию. Проверьте соединение и повторите."
        onRetry={() => void refetch()}
      />
    );
  }

  if (!user) {
    return (
      <Navigate to={STATIC_ROUTES.LOGIN} replace state={{ from: location }} />
    );
  }

  // Это удобство навигации. Доступ к данным всё равно проверяет Go на каждом запросе.
  if (roles && !hasRole(user, roles)) {
    return <Navigate to={STATIC_ROUTES.FORBIDDEN} replace />;
  }

  return <Outlet />;
};
