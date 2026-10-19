import { Outlet } from 'react-router';

import { useCurrentUser } from '@/entities/user';

import { Button } from '@/shared/ui';

export const AdminAccess = () => {
  const { data: user, isPending, isError, refetch } = useCurrentUser();
  const next = `${window.location.pathname}${window.location.search}${window.location.hash}`;

  let content;
  if (isPending) {
    content = <p role="status">Проверяем доступ…</p>;
  } else if (isError) {
    content = (
      <section aria-labelledby="access-error">
        <h1 id="access-error" className="mb-4 text-2xl font-semibold">
          Не удалось проверить доступ
        </h1>
        <p role="alert" className="mb-4">
          Проверьте подключение к интернету и попробуйте снова.
        </p>
        <Button onClick={() => void refetch()}>Повторить</Button>
      </section>
    );
  } else if (!user) {
    content = (
      <section aria-labelledby="login-title">
        <h1 id="login-title" className="mb-4 text-2xl font-semibold">
          Вход в панель администратора
        </h1>
        <p className="mb-4">Войдите с учётной записью администратора.</p>
        <Button
          render={<a href={`/auth/login?next=${encodeURIComponent(next)}`} />}
        >
          Войти
        </Button>
      </section>
    );
  } else if (!user.roles.includes('admin')) {
    // Проверка нужна для навигации. Защиту API обеспечивает gateway и сервис.
    content = (
      <section aria-labelledby="forbidden-title">
        <h1 id="forbidden-title" className="mb-4 text-2xl font-semibold">
          Доступ ограничен
        </h1>
        <p>Эта страница доступна только администраторам.</p>
      </section>
    );
  } else {
    content = <Outlet />;
  }

  return (
    <div className="min-h-svh bg-background text-foreground">
      <header className="border-b">
        <nav
          aria-label="Основная навигация"
          className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-4 px-6 py-4"
        >
          <a
            href="/admin/"
            className="font-semibold underline-offset-4 hover:underline"
          >
            Администрирование
          </a>
          <a href="/" className="underline underline-offset-4">
            Открыть сайт
          </a>
        </nav>
      </header>
      <main className="mx-auto max-w-5xl px-6 py-10">{content}</main>
    </div>
  );
};
