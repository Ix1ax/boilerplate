import { USER_ROLES } from './constants';

export type UserRole = (typeof USER_ROLES)[number];

export type { User } from '@starter/contracts';
