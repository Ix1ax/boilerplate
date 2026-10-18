import { useCallback, useState } from 'react';
import { RouterProvider } from 'react-router';

import { LoadingPage } from '@/pages/loading';

import { ErrorBoundary } from '@/shared/ui';

import { Providers } from './providers';
import { router } from './router';

export const App = () => {
  const [introVisible, setIntroVisible] = useState(true);
  const finishIntro = useCallback(() => setIntroVisible(false), []);

  return (
    <Providers showWidgets={!introVisible}>
      <ErrorBoundary>
        {introVisible ? (
          <LoadingPage onComplete={finishIntro} />
        ) : (
          <RouterProvider router={router} />
        )}
      </ErrorBoundary>
    </Providers>
  );
};
