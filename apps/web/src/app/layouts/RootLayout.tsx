import { Outlet, useNavigation } from 'react-router';

import { Header } from '@/widgets/header';

export const RootLayout = () => {
  const navigation = useNavigation();
  const isNavigating = navigation.state === 'loading';

  return (
    <div className="min-h-screen">
      <a
        href="#main-content"
        className="sr-only focus-visible:not-sr-only focus-visible:fixed focus-visible:top-2 focus-visible:left-2 focus-visible:z-50 focus-visible:rounded-md focus-visible:bg-primary focus-visible:px-4 focus-visible:py-2 focus-visible:text-primary-foreground"
      >
        Перейти к содержимому
      </a>

      {isNavigating && (
        <div
          role="progressbar"
          aria-label="Загрузка страницы"
          className="fixed inset-x-0 top-0 h-0.5 animate-pulse bg-current"
        />
      )}

      <Header />

      <main id="main-content" className="p-6">
        <Outlet />
      </main>
    </div>
  );
};
