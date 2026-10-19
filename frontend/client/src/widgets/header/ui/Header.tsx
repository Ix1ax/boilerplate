import { useRef, useState } from 'react';
import { Link, NavLink } from 'react-router';
import { cn } from 'cn';
import {
  BuildingIcon,
  LogOutIcon,
  MoonIcon,
  PlusIcon,
  SettingsIcon,
  ShieldCheckIcon,
  SunIcon,
  TrophyIcon,
} from 'lucide-react';

import { useLogout } from '@/features/auth';
import { NotificationsBell } from '@/features/notifications';
import { useTheme } from '@/features/theme';

import { hasRole, useCurrentUser } from '@/entities/user';

import { STATIC_ROUTES } from '@/shared/constants';
import { useEventListener } from '@/shared/hooks';
import { initialOf } from '@/shared/lib';
import { Lift } from '@/shared/motion';
import {
  BrandMark,
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/shared/ui';

const NAV = [
  { to: STATIC_ROUTES.EVENTS, label: 'Найти встречу' },
  { to: STATIC_ROUTES.MAP, label: 'Карта' },
  { to: STATIC_ROUTES.MY, label: 'Мой спорт' },
  { to: STATIC_ROUTES.COMPANY, label: 'Собрать компанию' },
];

// На лендинге часть пунктов — якоря его же секций.
const LANDING_NAV = [
  { to: STATIC_ROUTES.EVENTS, label: 'Встречи' },
  { to: STATIC_ROUTES.COMPANY, label: 'Собрать компанию' },
  { to: '#start', label: 'Новичкам' },
  { to: '#clubs', label: 'Клубам' },
];

interface HeaderProps {
  /** dark — прозрачная шапка поверх тёмной первой секции лендинга. */
  tone?: 'light' | 'dark';
}

const UserMenu = ({ name, isAdmin }: { name: string; isAdmin: boolean }) => {
  const logout = useLogout();
  const { resolvedTheme, setTheme } = useTheme();
  const isDark = resolvedTheme === 'dark';

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={`Профиль: ${name}`}
        className="grid size-12 shrink-0 place-items-center rounded-full bg-gold text-lg font-extrabold text-ink"
      >
        {initialOf(name)}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={8}>
        <DropdownMenuGroup>
          <DropdownMenuLabel>{name}</DropdownMenuLabel>
          <DropdownMenuItem render={<Link to={STATIC_ROUTES.MY} />}>
            <TrophyIcon aria-hidden />
            Мой спорт
          </DropdownMenuItem>
          <DropdownMenuItem render={<Link to={STATIC_ROUTES.CLUB} />}>
            <BuildingIcon aria-hidden />
            Кабинет клуба
          </DropdownMenuItem>
          {isAdmin && (
            <DropdownMenuItem render={<a href="/admin/" />}>
              <ShieldCheckIcon aria-hidden />
              Администрирование
            </DropdownMenuItem>
          )}
          <DropdownMenuItem render={<Link to={STATIC_ROUTES.SETTINGS} />}>
            <SettingsIcon aria-hidden />
            Профиль и настройки
          </DropdownMenuItem>
          <DropdownMenuItem
            closeOnClick={false}
            onClick={() => setTheme(isDark ? 'light' : 'dark')}
          >
            {isDark ? <SunIcon aria-hidden /> : <MoonIcon aria-hidden />}
            {isDark ? 'Светлая тема' : 'Тёмная тема'}
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          disabled={logout.isPending}
          onClick={() => logout.mutate()}
        >
          <LogOutIcon aria-hidden />
          Выйти
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

/** Шапка приложения: знак, три раздела, «Создать встречу» и меню профиля. */
export const Header = ({ tone = 'light' }: HeaderProps) => {
  const { data: user, isLoading } = useCurrentUser();
  const dark = tone === 'dark';

  // Шапка прячется при скролле вниз и возвращается, как только пользователь начинает скроллить вверх.
  const [hidden, setHidden] = useState(false);
  const lastY = useRef(0);
  useEventListener('scroll', () => {
    const y = window.scrollY;
    const delta = y - lastY.current;
    if (y < 80) {
      setHidden(false);
    } else if (Math.abs(delta) > 4) {
      setHidden(delta > 0);
    }
    lastY.current = y;
  });

  return (
    <header
      className={cn(
        'sticky top-0 z-30 transition-transform duration-300 will-change-transform',
        hidden && '-translate-y-full',
        dark
          ? 'bg-ink text-paper'
          : 'border-b border-line bg-card dark:border-border',
      )}
    >
      <div
        className={cn(
          'mx-auto flex max-w-[1280px] flex-wrap items-center gap-x-6 gap-y-3 px-4 sm:px-8',
          dark ? 'max-w-[1360px] py-5' : 'py-3',
        )}
      >
        <Link to={STATIC_ROUTES.HOME} className="flex min-h-11 items-center">
          <BrandMark
            tone={dark ? 'dark' : 'light'}
            tagline={dark ? 'компания для спорта рядом' : undefined}
          />
          {/* Доступное имя должно начинаться с видимого текста логотипа (WCAG 2.5.3), поэтому
              пояснение — скрытым текстом после него, а не aria-label. */}
          <span className="sr-only"> — на главную</span>
        </Link>

        <nav
          aria-label="Основная навигация"
          className="order-last flex w-full flex-wrap gap-1 md:order-none md:w-auto md:flex-1"
        >
          {dark &&
            LANDING_NAV.map((item) =>
              item.to.startsWith('#') ? (
                <a
                  key={item.to}
                  href={item.to}
                  className="flex min-h-11 items-center rounded-full px-4 font-semibold transition-colors hover:bg-paper/10"
                >
                  {item.label}
                </a>
              ) : (
                <Link
                  key={item.to}
                  to={item.to}
                  className="flex min-h-11 items-center rounded-full px-4 font-semibold transition-colors hover:bg-paper/10"
                >
                  {item.label}
                </Link>
              ),
            )}
          {!dark &&
            NAV.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  cn(
                    'flex min-h-11 items-center rounded-full px-4 font-semibold transition-colors',
                    isActive
                      ? 'bg-ink font-bold text-paper dark:bg-paper dark:text-ink'
                      : 'hover:bg-muted',
                  )
                }
              >
                {item.label}
              </NavLink>
            ))}
        </nav>

        <div className="ml-auto flex items-center gap-2.5 md:ml-0">
          {user ? (
            <>
              <Button
                variant={dark ? 'gold' : 'default'}
                className="max-sm:hidden"
                render={<Link to={STATIC_ROUTES.EVENT_NEW} />}
              >
                <PlusIcon aria-hidden />
                Создать встречу
              </Button>
              <NotificationsBell />
              <UserMenu
                name={user.name || user.email}
                isAdmin={hasRole(user, ['admin'])}
              />
            </>
          ) : (
            !isLoading && (
              <>
                <Lift>
                  <Button
                    variant={dark ? 'outline-light' : 'outline'}
                    render={<Link to={STATIC_ROUTES.LOGIN} />}
                  >
                    Войти
                  </Button>
                </Lift>
                {dark && (
                  <Lift className="inline-flex max-sm:hidden">
                    <Button
                      variant="gold"
                      render={<Link to={STATIC_ROUTES.EVENTS} />}
                    >
                      Найти встречу
                    </Button>
                  </Lift>
                )}
              </>
            )
          )}
        </div>
      </div>
    </header>
  );
};
