import type { Profile as ProfileDto } from '@starter/contracts';

import type { Camelize } from '@/shared/api';

export type Profile = Camelize<ProfileDto>;
