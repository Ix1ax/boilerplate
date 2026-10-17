import { Link } from 'react-router';

import { STATIC_ROUTES } from '@/shared/constants';
import { buttonVariants } from '@/shared/ui';

export const ForbiddenPage = () => {
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
      <div className="space-y-1">
        <h1 className="text-3xl font-semibold">403</h1>
        <p className="text-sm text-muted-foreground">Доступ запрещён</p>
      </div>
      <Link
        to={STATIC_ROUTES.HOME}
        className={buttonVariants({ variant: 'outline' })}
      >
        На главную
      </Link>
    </div>
  );
};
