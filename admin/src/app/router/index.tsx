import { createBrowserRouter } from 'react-router';

import { AdminPage } from '@/pages/admin';
import { NotFoundPage } from '@/pages/not-found';

import { AdminAccess } from '@/features/admin-access';

export const router = createBrowserRouter(
  [
    {
      element: <AdminAccess />,
      children: [
        { index: true, element: <AdminPage /> },
        { path: '*', element: <NotFoundPage /> },
      ],
    },
  ],
  { basename: '/admin' },
);
