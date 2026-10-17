export const STATIC_ROUTES = {
  HOME: '/',
  PROFILE: '/profile',
  LOGIN: '/login',
  DASHBOARD: '/dashboard',
  FORBIDDEN: '/403',
} as const;

export const DYNAMIC_ROUTES = {
  POST: (postId: number) => `/post/${postId}`,
} as const;
