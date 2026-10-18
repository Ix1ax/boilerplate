import { Link } from 'react-router';

import { useLogout } from '@/features/auth';
import { NotificationsBell } from '@/features/notifications';
import { ThemeToggle } from '@/features/theme';

import { useCurrentUser } from '@/entities/user';

import { STATIC_ROUTES } from '@/shared/constants';
import { Button, Container } from '@/shared/ui';

export const Header = () => {
  const { data: user, isLoading } = useCurrentUser();
  const logout = useLogout();

  return (
    <header className="border-b">
      <Container className="flex items-center gap-4 py-3">
        <Link to={STATIC_ROUTES.HOME} className="font-semibold">
          Hackathon
        </Link>

        <span className="flex-1" />

        <ThemeToggle />

        {user ? (
          <div className="flex items-center gap-2">
            <NotificationsBell />
            <span className="text-sm">{user.name}</span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => logout.mutate()}
              disabled={logout.isPending}
            >
              Выйти
            </Button>
          </div>
        ) : (
          !isLoading && (
            <Button
              variant="outline"
              size="sm"
              render={<Link to={STATIC_ROUTES.LOGIN} />}
            >
              Войти
            </Button>
          )
        )}
      </Container>
    </header>
  );
};
