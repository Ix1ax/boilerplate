import { Outlet, useLocation, useMatch, useNavigation } from 'react-router';
import { motion } from 'motion/react';

import { Footer } from '@/widgets/footer';
import { Header } from '@/widgets/header';

import { STATIC_ROUTES } from '@/shared/constants';
import { DURATION, EASING } from '@/shared/motion';

export const RootLayout = () => {
  const navigation = useNavigation();
  const { pathname } = useLocation();
  const isNavigating = navigation.state === 'loading';
  // Лендинг открывается тёмной секцией: шапка и подвал — в его цветах.
  const isLanding = useMatch(STATIC_ROUTES.HOME) !== null;
  // Экран входа — отдельный двухколоночный макет без шапки приложения.
  const isAuth = useMatch(STATIC_ROUTES.LOGIN) !== null;

  return (
    <div className="flex min-h-screen flex-col">
      <a
        href="#main-content"
        className="sr-only focus-visible:not-sr-only focus-visible:fixed focus-visible:top-2 focus-visible:left-2 focus-visible:z-50 focus-visible:rounded-full focus-visible:bg-primary focus-visible:px-5 focus-visible:py-3 focus-visible:font-bold focus-visible:text-primary-foreground"
      >
        Перейти к содержимому
      </a>

      {isNavigating && (
        <div
          role="progressbar"
          aria-label="Загрузка страницы"
          className="fixed inset-x-0 top-0 z-50 h-1 animate-pulse bg-gold"
        />
      )}

      {!isAuth && <Header tone={isLanding ? 'dark' : 'light'} />}

      <main id="main-content" className="flex flex-1 flex-col">
        {/* Смена экрана — быстрое появление; transform не используем, чтобы не ломать sticky. */}
        <motion.div
          key={pathname}
          className="flex flex-1 flex-col"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: DURATION.quick, ease: EASING.out }}
        >
          <Outlet />
        </motion.div>
      </main>

      {!isAuth && <Footer tone={isLanding ? 'dark' : 'light'} />}
    </div>
  );
};
