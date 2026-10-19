import { createBrowserRouter, Navigate } from 'react-router';

import { DashboardLayout, RootLayout } from '@/app/layouts';

import { ErrorPage } from '@/pages/error';
import { ForbiddenPage } from '@/pages/forbidden';
import { NotFoundPage } from '@/pages/not-found';

import { DYNAMIC_ROUTES, STATIC_ROUTES } from '@/shared/constants';
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
          { path: STATIC_ROUTES.ABOUT, lazy: () => import('@/pages/about') },
          { path: STATIC_ROUTES.FORBIDDEN, element: <ForbiddenPage /> },
          {
            // Весь API требует сессию: гостя отправляем на вход и возвращаем обратно.
            element: <ProtectedRoute />,
            children: [
              {
                path: STATIC_ROUTES.COMPANY,
                lazy: () => import('@/pages/company'),
              },
              { path: STATIC_ROUTES.CLUB, lazy: () => import('@/pages/club') },
              {
                path: STATIC_ROUTES.EVENT_NEW,
                lazy: () => import('@/pages/event-new'),
              },
              {
                path: DYNAMIC_ROUTES.EVENT_MANAGE(':eventId'),
                lazy: () => import('@/pages/event-manage'),
              },
              {
                path: DYNAMIC_ROUTES.ORGANIZATION(':orgId'),
                lazy: () => import('@/pages/organization'),
              },
              {
                path: DYNAMIC_ROUTES.PERSON(':userId'),
                lazy: () => import('@/pages/person'),
              },
              {
                path: STATIC_ROUTES.EVENTS,
                lazy: () => import('@/pages/events'),
              },
              {
                path: DYNAMIC_ROUTES.EVENT(':eventId'),
                lazy: () => import('@/pages/event'),
              },
              {
                path: STATIC_ROUTES.MY,
                lazy: () => import('@/pages/my-sport'),
              },
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
                    path: STATIC_ROUTES.DASHBOARD_TASKS,
                    lazy: () => import('@/pages/dashboard-tasks'),
                  },
                  {
                    path: STATIC_ROUTES.DASHBOARD_FILES,
                    lazy: () => import('@/pages/dashboard-files'),
                  },
                  {
                    path: STATIC_ROUTES.DASHBOARD_ASSISTANT,
                    lazy: () => import('@/pages/dashboard-assistant'),
                  },
                  {
                    path: STATIC_ROUTES.DASHBOARD_LIVE,
                    lazy: () => import('@/pages/dashboard-live'),
                  },
                  {
                    element: <ProtectedRoute roles={['admin']} />,
                    children: [
                      {
                        path: STATIC_ROUTES.DASHBOARD_ADMIN,
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
