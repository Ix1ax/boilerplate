import type { Transition } from 'motion/react';

import { DURATION, EASING } from './tokens';

/** Готовые пресеты переходов поверх токенов. */
export const TRANSITION = {
  fast: { duration: DURATION.quick, ease: EASING.out },
  base: { duration: DURATION.base, ease: EASING.out },
  slow: { duration: DURATION.slow, ease: EASING.out },
  cine: { duration: DURATION.cine, ease: EASING.out },
  emphasized: { duration: DURATION.slow, ease: EASING.emphasized },
} satisfies Record<string, Transition>;
