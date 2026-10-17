import { USER_ROLES } from './constants';

export type UserRole = (typeof USER_ROLES)[number];

export type User = {
  id: string;
  name: string;
  avatarUrl: string | null;
  role: UserRole;
};
