export const STATIC_ROUTES = {
  HOME: '/',
  PROFILE: '/profile',
  LOGIN: '/login',
  DASHBOARD: '/dashboard',
  DASHBOARD_TASKS: '/dashboard/tasks',
  DASHBOARD_FILES: '/dashboard/files',
  FORBIDDEN: '/403',
} as const;

export const DYNAMIC_ROUTES = {
  POST: (postId: number) => `/post/${postId}`,
} as const;
