import type { Profile as ProfileDto } from '@starter/contracts';

import type { Camelize } from '@/shared/api';

export type Profile = Camelize<ProfileDto> & {
  interests: string[];
  preferences?: {
    notifyReminders: boolean;
    notifyDecisions: boolean;
    notifyReplies: boolean;
    largeText: boolean;
    reducedMotion: boolean;
  };
};
