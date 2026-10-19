import type { Transition } from 'motion/react';

import { DURATION, EASING } from './tokens';

/** Готовые пресеты переходов поверх токенов. */
export const TRANSITION = {
  fast: { duration: DURATION.fast, ease: EASING.standard },
  base: { duration: DURATION.base, ease: EASING.standard },
  emphasized: { duration: DURATION.slow, ease: EASING.emphasized },
} satisfies Record<string, Transition>;
