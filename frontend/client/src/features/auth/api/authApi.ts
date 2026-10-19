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

export interface AuthProviders {
  sso: { enabled: boolean; title: string };
  local: { enabled: boolean };
}
export const getAuthProviders = () =>
  request<AuthProviders>({ url: '/auth/providers', baseURL: '/' });
export const localLogin = (data: { email: string; password: string }) =>
  request({ url: '/auth/local/login', baseURL: '/', method: 'POST', data });
export const localRegister = (data: {
  name: string;
  email: string;
  password: string;
}) =>
  request({ url: '/auth/local/register', baseURL: '/', method: 'POST', data });
