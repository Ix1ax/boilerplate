import { type HTMLMotionProps, motion } from 'motion/react';

import {
  fadeIn,
  scaleIn,
  slideUp,
  staggerContainer,
  staggerItem,
} from './variants';

type MotionDivProps = HTMLMotionProps<'div'>;

/** Плавное появление по прозрачности. */
export const FadeIn = (props: MotionDivProps) => (
  <motion.div variants={fadeIn} initial="hidden" animate="visible" {...props} />
);

/** Появление снизу вверх. */
export const SlideUp = (props: MotionDivProps) => (
  <motion.div
    variants={slideUp}
    initial="hidden"
    animate="visible"
    {...props}
  />
);

/** Появление с масштабом. */
export const ScaleIn = (props: MotionDivProps) => (
  <motion.div
    variants={scaleIn}
    initial="hidden"
    animate="visible"
    {...props}
  />
);

/** Контейнер для поочерёдного появления детей (оборачивай элементы в StaggerItem). */
export const Stagger = (props: MotionDivProps) => (
  <motion.div
    variants={staggerContainer}
    initial="hidden"
    animate="visible"
    {...props}
  />
);

/** Элемент списка внутри Stagger (анимацию запускает контейнер). */
export const StaggerItem = (props: MotionDivProps) => (
  <motion.div variants={staggerItem} {...props} />
);
