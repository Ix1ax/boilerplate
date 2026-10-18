import { useEffect } from 'react';
import { isRouteErrorResponse, Link, useRouteError } from 'react-router';

import { STATIC_ROUTES } from '@/shared/constants';
import { logger } from '@/shared/lib';
import { buttonVariants } from '@/shared/ui';

type ResolvedError = {
  title: string;
  message: string;
  unexpected: boolean;
};

const resolveError = (error: unknown): ResolvedError => {
  if (isRouteErrorResponse(error)) {
    return {
      title: `Ошибка ${error.status}`,
      message: error.statusText || 'Не удалось загрузить страницу',
      unexpected: false,
    };
  }

  if (error instanceof Error) {
    return {
      title: 'Что-то пошло не так',
      message: error.message,
      unexpected: true,
    };
  }

  return {
    title: 'Что-то пошло не так',
    message: 'Неизвестная ошибка',
    unexpected: true,
  };
};

export const ErrorPage = () => {
  const error = useRouteError();
  const { title, message, unexpected } = resolveError(error);

  useEffect(() => {
    if (unexpected) logger.error('Ошибка маршрута', error);
  }, [error, unexpected]);

  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold">{title}</h1>
        <p className="text-sm text-muted-foreground">{message}</p>
      </div>
      <Link to={STATIC_ROUTES.HOME} className={buttonVariants()}>
        На главную
      </Link>
    </div>
  );
};
