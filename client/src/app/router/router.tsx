import { createBrowserRouter, Navigate } from 'react-router';

import { DashboardLayout, RootLayout } from '@/app/layouts';

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
                element: <DashboardLayout />,
                children: [
                  {
                    index: true,
                    element: (
                      <Navigate to={STATIC_ROUTES.DASHBOARD_TASKS} replace />
                    ),
                  },
                  {
                    path: 'tasks',
                    lazy: () => import('@/pages/dashboard-tasks'),
                  },
                  {
                    path: 'files',
                    lazy: () => import('@/pages/dashboard-files'),
                  },
                  {
                    path: 'assistant',
                    lazy: () => import('@/pages/dashboard-assistant'),
                  },
                  {
                    element: <ProtectedRoute roles={['admin']} />,
                    children: [
                      {
                        path: 'admin',
                        lazy: () => import('@/pages/dashboard-admin'),
                      },
                    ],
                  },
                ],
              },
            ],
          },
          { path: '*', element: <NotFoundPage /> },
        ],
      },
    ],
  },
]);
