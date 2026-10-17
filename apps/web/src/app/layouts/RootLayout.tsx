import { Outlet, useNavigation } from 'react-router';

import { Header } from '@/widgets/header';

export const RootLayout = () => {
  const navigation = useNavigation();
  const isNavigating = navigation.state === 'loading';

  return (
    <div className="min-h-screen">
      {isNavigating && (
        <div
          role="progressbar"
          aria-label="Загрузка страницы"
          className="fixed inset-x-0 top-0 h-0.5 animate-pulse bg-current"
        />
      )}

      <Header />

      <main className="p-6">
        <Outlet />
      </main>
    </div>
  );
};
