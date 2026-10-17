import type { ReactNode } from 'react';
import { MotionConfig } from 'motion/react';

import { TRANSITION } from '@/shared/motion';

interface MotionProviderProps {
  children?: ReactNode;
  forceReducedMotion?: boolean;
}

export const MotionProvider = ({
  children,
  forceReducedMotion,
}: MotionProviderProps) => {
  return (
    <MotionConfig
      reducedMotion={forceReducedMotion ? 'always' : 'user'}
      transition={TRANSITION.base}
    >
      {children}
    </MotionConfig>
  );
};
