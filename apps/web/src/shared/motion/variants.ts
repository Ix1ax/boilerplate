import type { Variants } from 'motion/react';

import { TRANSITION } from './transitions';

/** Появление по прозрачности. */
export const fadeIn: Variants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1, transition: TRANSITION.base },
};

/** Появление снизу вверх. */
export const slideUp: Variants = {
  hidden: { opacity: 0, y: 12 },
  visible: { opacity: 1, y: 0, transition: TRANSITION.base },
};

/** Появление с лёгким масштабом. */
export const scaleIn: Variants = {
  hidden: { opacity: 0, scale: 0.96 },
  visible: { opacity: 1, scale: 1, transition: TRANSITION.base },
};

/** Контейнер для поочерёдного появления детей (списки). */
export const staggerContainer: Variants = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.06 } },
};

/** Элемент внутри staggerContainer (initial/animate наследуются от контейнера). */
export const staggerItem: Variants = slideUp;
