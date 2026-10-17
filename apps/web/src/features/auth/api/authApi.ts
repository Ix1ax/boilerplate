import { request, type Tokens } from '@/shared/api';

import type { LoginCredentials } from '../model/types';

// '/auth/login' входит в NO_REFRESH_URLS → refresh-интерсептор его пропускает.
export const login = (credentials: LoginCredentials) =>
  request<Tokens>({ url: '/auth/login', method: 'POST', data: credentials });

export const logout = () =>
  request<void>({ url: '/auth/logout', method: 'POST' });
