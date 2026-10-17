import { request } from '@/shared/api';

import type { User } from '../model';

export const getMe = () => request<User>({ url: '/me' });
export const getUser = (id: string) => request<User>({ url: `/users/${id}` });
export const getUsers = () => request<User[]>({ url: '/users' });
