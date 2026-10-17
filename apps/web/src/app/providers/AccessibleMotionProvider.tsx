import type { ReactNode } from 'react';
import { useAccessibility } from 'open-accessibility-widget/react';

import { MotionProvider } from './MotionProvider';

interface AccessibleMotionProviderProps {
  children?: ReactNode;
}

export const AccessibleMotionProvider = ({
  children,
}: AccessibleMotionProviderProps) => {
  const { settings } = useAccessibility();

  return (
    <MotionProvider forceReducedMotion={settings.stopAnimations}>
      {children}
    </MotionProvider>
  );
};
