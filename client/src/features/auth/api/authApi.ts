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

export type DemoAccount = {
  key: string;
  name: string;
  /** Роль простыми словами. */
  role: string;
  /** Что проверить под этим аккаунтом. */
  hint: string;
};

// Маршрут есть только при DEMO_LOGIN=true; иначе 404 — показываем обычный вход.
export const getDemoAccounts = () =>
  request<{ items: DemoAccount[] }>({ url: '/auth/demo', baseURL: '/' })
    .then((data) => data.items)
    .catch(() => [] as DemoAccount[]);

export const demoLogin = (key: string) =>
  request<{ key: string; name: string }>({
    url: `/auth/demo/${encodeURIComponent(key)}`,
    baseURL: '/',
    method: 'POST',
  });
