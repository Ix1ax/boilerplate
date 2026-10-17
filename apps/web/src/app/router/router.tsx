import { createBrowserRouter } from 'react-router';

import { RootLayout } from '@/app/layouts';

import { ErrorPage } from '@/pages/error';
import { ForbiddenPage } from '@/pages/forbidden';
import { NotFoundPage } from '@/pages/not-found';

import { STATIC_ROUTES } from '@/shared/constants';
import { LoaderScreen } from '@/shared/ui';

import { ProtectedRoute } from './ProtectedRoute';

export const router = createBrowserRouter([
  {
    element: <RootLayout />,
    errorElement: <ErrorPage />,
    hydrateFallbackElement: <LoaderScreen />,
    children: [
      {
        errorElement: <ErrorPage />,
        children: [
          { index: true, lazy: () => import('@/pages/home') },
          { path: STATIC_ROUTES.LOGIN, lazy: () => import('@/pages/login') },
          { path: STATIC_ROUTES.FORBIDDEN, element: <ForbiddenPage /> },
          {
            element: <ProtectedRoute />,
            children: [
              {
                path: STATIC_ROUTES.DASHBOARD,
                lazy: () => import('@/pages/dashboard'),
              },
            ],
          },
          { path: '*', element: <NotFoundPage /> },
        ],
      },
    ],
  },
]);
