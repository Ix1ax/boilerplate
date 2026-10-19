export const STATIC_ROUTES = {
  HOME: '/',
  LOGIN: '/login',
  EVENTS: '/events',
  EVENT_NEW: '/events/new',
  MAP: '/map',
  MY: '/my',
  SETTINGS: '/settings',
  COMPANY: '/company',
  CLUB: '/club',
  ABOUT: '/about',
  DASHBOARD: '/dashboard',
  DASHBOARD_TASKS: '/dashboard/tasks',
  DASHBOARD_FILES: '/dashboard/files',
  DASHBOARD_ASSISTANT: '/dashboard/assistant',
  DASHBOARD_ADMIN: '/dashboard/admin',
  DASHBOARD_LIVE: '/dashboard/live',
  FORBIDDEN: '/403',
} as const;

export const DYNAMIC_ROUTES = {
  EVENT: (eventId: string) => `/events/${eventId}`,
  EVENT_MANAGE: (eventId: string) => `/events/${eventId}/manage`,
  ORGANIZATION: (orgId: string) => `/organizations/${orgId}`,
  PERSON: (userId: string) => `/people/${userId}`,
} as const;
