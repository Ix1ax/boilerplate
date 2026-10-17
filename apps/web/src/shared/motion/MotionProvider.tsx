import type { ReactNode } from 'react';
import { MotionConfig } from 'motion/react';

import { TRANSITION } from './transitions';

interface MotionProviderProps {
  children?: ReactNode;
}

/**
 * Глобальная конфигурация анимаций. `reducedMotion="user"` автоматически
 * уважает системную настройку prefers-reduced-motion.
 */
export const MotionProvider = ({ children }: MotionProviderProps) => {
  return (
    <MotionConfig reducedMotion="user" transition={TRANSITION.base}>
      {children}
    </MotionConfig>
  );
};
