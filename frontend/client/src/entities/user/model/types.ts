import type { User as UserDto } from '@starter/contracts';

import type { Camelize } from '@/shared/api';

import { USER_ROLES } from './constants';

export type UserRole = (typeof USER_ROLES)[number];

export type User = Camelize<UserDto>;
