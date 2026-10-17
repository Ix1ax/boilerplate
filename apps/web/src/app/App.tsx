import { RouterProvider } from 'react-router';

import { ErrorBoundary } from '@/shared/ui';

import { Providers } from './providers';
import { router } from './router';

export const App = () => {
  return (
    <Providers>
      <ErrorBoundary>
        <RouterProvider router={router} />
      </ErrorBoundary>
    </Providers>
  );
};
