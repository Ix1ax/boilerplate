import type { ReactNode } from 'react';
import { motion } from 'motion/react';

import { DURATION, EASING } from './tokens';

type Tag = 'div' | 'li' | 'section' | 'span' | 'p' | 'h2';

interface RevealProps {
  children?: ReactNode;
  className?: string;
  as?: Tag;
  /** Задержка в секундах; для списков — `staggerDelay(index)`. */
  delay?: number;
  /** Смещение снизу в px. */
  y?: number;
  /** Продолжительность: `cine` — для заголовков лендинга. */
  duration?: number;
  /** Появление сразу при загрузке, а не при входе во вьюпорт. */
  immediate?: boolean;
  /** Начальный масштаб (радар: 0.94 → 1). */
  scale?: number;
}

/**
 * Появление снизу вверх при входе во вьюпорт (один раз).
 * При «Меньше движения» остаётся только плавная смена прозрачности.
 */
export const Reveal = ({
  children,
  className,
  as = 'div',
  delay = 0,
  y = 24,
  duration = DURATION.slow,
  immediate,
  scale,
}: RevealProps) => {
  const Tag = motion[as] as typeof motion.div;
  const hidden = { opacity: 0, y, ...(scale ? { scale } : {}) };
  const shown = { opacity: 1, y: 0, ...(scale ? { scale: 1 } : {}) };
  const transition = { duration, ease: EASING.out, delay };

  return (
    <Tag
      className={className}
      initial={hidden}
      {...(immediate
        ? { animate: shown }
        : {
            whileInView: shown,
            viewport: { once: true, amount: 0.2, margin: '0px 0px -8% 0px' },
          })}
      transition={transition}
    >
      {children}
    </Tag>
  );
};
