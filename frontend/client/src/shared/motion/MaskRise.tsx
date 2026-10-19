import { type ReactNode, useRef } from 'react';
import { cn } from 'cn';
import { motion, useInView } from 'motion/react';

import { DURATION, EASING } from './tokens';

interface MaskRiseProps {
  children: ReactNode;
  className?: string;
  delay?: number;
  duration?: number;
  /** Запускать при входе во вьюпорт, а не при загрузке. */
  inView?: boolean;
}

/**
 * «Mask rise» заголовка: строка выезжает из-под маски.
 * Маска режет только по вертикали — широкие буквы не обрезаются справа;
 * нижний отступ компенсирует обрезку хвостов букв при плотном line-height.
 * Появление отслеживаем по внешней обёртке: внутренний span сам скрыт маской.
 */
export const MaskRise = ({
  children,
  className,
  delay = 0,
  duration = DURATION.cine,
  inView,
}: MaskRiseProps) => {
  const ref = useRef<HTMLSpanElement>(null);
  const seen = useInView(ref, { once: true, amount: 0.3 });
  const shown = inView ? seen : true;

  return (
    <span
      ref={ref}
      className={cn(
        'block overflow-x-visible overflow-y-clip pt-[0.04em] pb-[0.14em] -mb-[0.14em]',
        className,
      )}
    >
      <motion.span
        className="block"
        initial={{ y: '108%' }}
        animate={{ y: shown ? 0 : '108%' }}
        transition={{ duration, ease: EASING.out, delay }}
      >
        {children}
      </motion.span>
    </span>
  );
};
