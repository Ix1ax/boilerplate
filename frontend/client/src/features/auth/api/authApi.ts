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

export const changeLocalPassword = (data: {
  current_password: string;
  new_password: string;
}) =>
  request<void>({
    url: '/auth/local/password',
    baseURL: '/',
    method: 'POST',
    data,
  });
