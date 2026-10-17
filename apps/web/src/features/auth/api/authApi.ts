import { request } from '@/shared/api';

export const login = (next: string) => {
  window.location.assign(`/auth/login?next=${encodeURIComponent(next)}`);
};

export const logout = () =>
  request<{ url: string }>({
    url: '/auth/logout',
    baseURL: '/',
    method: 'POST',
  });
