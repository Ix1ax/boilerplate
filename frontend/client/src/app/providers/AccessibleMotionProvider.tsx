import { type ReactNode, useEffect, useState } from 'react';
import { useAccessibility } from 'open-accessibility-widget/react';

import { MotionProvider } from './MotionProvider';

interface AccessibleMotionProviderProps {
  children?: ReactNode;
}

export const AccessibleMotionProvider = ({
  children,
}: AccessibleMotionProviderProps) => {
  const { settings } = useAccessibility();
  const [profileReduced, setProfileReduced] = useState(
    document.documentElement.dataset.reducedMotion === 'true',
  );
  useEffect(() => {
    const sync = () =>
      setProfileReduced(
        document.documentElement.dataset.reducedMotion === 'true',
      );
    window.addEventListener('profile-preferences-changed', sync);
    return () =>
      window.removeEventListener('profile-preferences-changed', sync);
  }, []);

  return (
    <MotionProvider
      forceReducedMotion={settings.stopAnimations || profileReduced}
    >
      {children}
    </MotionProvider>
  );
};
