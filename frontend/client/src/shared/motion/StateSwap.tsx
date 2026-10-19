import type { ReactNode } from 'react';
import { AnimatePresence, motion } from 'motion/react';

import { SPRING_POP } from './tokens';
import { TRANSITION } from './transitions';

interface StateSwapProps {
  /** Ключ состояния: при смене старое уходит, новое появляется. */
  stateKey: string;
  children: ReactNode;
  className?: string;
}

/** Смена статуса (записаться → записан → ожидает решения): ≤ 280 мс, без задержки кнопки. */
export const StateSwap = ({
  stateKey,
  children,
  className,
}: StateSwapProps) => (
  <AnimatePresence mode="wait" initial={false}>
    <motion.div
      key={stateKey}
      className={className}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -8 }}
      transition={TRANSITION.base}
    >
      {children}
    </motion.div>
  </AnimatePresence>
);

/** Значок статуса «выскакивает» пружиной. */
export const PopIn = ({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) => (
  <motion.span
    className={className}
    initial={{ scale: 0 }}
    animate={{ scale: 1 }}
    transition={SPRING_POP}
    style={{ display: 'inline-block' }}
  >
    {children}
  </motion.span>
);

/** Индикатор загрузки на кнопке. */
export const Spin = ({ className }: { className?: string }) => (
  <motion.svg
    aria-hidden="true"
    width="22"
    height="22"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2.6"
    strokeLinecap="round"
    className={className}
    animate={{ rotate: 360 }}
    transition={{ duration: 0.8, ease: 'linear', repeat: Infinity }}
  >
    <path d="M12 3a9 9 0 1 0 9 9" />
  </motion.svg>
);
