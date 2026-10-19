import { useEffect } from 'react';

import { useMyProfile } from '@/entities/profile';
import { useCurrentUser } from '@/entities/user';

/** Настройки принадлежат аккаунту; после выхода удаляем их с документа. */
export const ProfilePreferencesSync = () => {
  const user = useCurrentUser();
  const profile = useMyProfile(Boolean(user.data));
  const largeText = Boolean(user.data && profile.data?.preferences?.largeText);
  const reducedMotion = Boolean(
    user.data && profile.data?.preferences?.reducedMotion,
  );
  useEffect(() => {
    document.documentElement.dataset.largeText = String(largeText);
    document.documentElement.dataset.reducedMotion = String(reducedMotion);
    window.dispatchEvent(
      new CustomEvent('profile-preferences-changed', {
        detail: { largeText, reducedMotion },
      }),
    );
  }, [largeText, reducedMotion]);
  return null;
};
