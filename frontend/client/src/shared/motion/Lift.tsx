import type { ReactNode } from 'react';
import { cn } from 'cn';
import { motion } from 'motion/react';

import { SPRING_POP } from './tokens';

interface LiftProps {
  children: ReactNode;
  /** По умолчанию `inline-flex` (кнопки); для карточек передайте `flex h-full w-full`. */
  className?: string;
  /** Насколько поднять при наведении, px. */
  distance?: number;
}

/**
 * Отклик на наведение и нажатие: элемент приподнимается пружиной, при нажатии
 * слегка сжимается. Вложенная ссылка или кнопка сохраняет свою семантику.
 */
export const Lift = ({ children, className, distance = 4 }: LiftProps) => (
  <motion.div
    className={cn('inline-flex', className)}
    whileHover={{
      y: -distance,
      boxShadow: '0 24px 44px -24px rgb(18 18 18 / 45%)',
    }}
    whileTap={{ scale: 0.97, y: 0 }}
    transition={SPRING_POP}
    style={{ borderRadius: 'inherit' }}
  >
    {children}
  </motion.div>
);
