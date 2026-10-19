import { type ReactNode, useRef } from 'react';
import {
  motion,
  useReducedMotion,
  useScroll,
  useTransform,
} from 'motion/react';

interface ParallaxProps {
  children?: ReactNode;
  className?: string;
  /** −0.3…0.3: доля скорости скролла; радар лендинга — −0.15. */
  speed?: number;
}

/** Смещение по скроллу. Без «Меньше движения» и анимаций системы не двигается. */
export const Parallax = ({
  children,
  className,
  speed = -0.15,
}: ParallaxProps) => {
  const ref = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotion();
  const { scrollYProgress } = useScroll({
    target: ref,
    offset: ['start end', 'end start'],
  });
  const y = useTransform(scrollYProgress, [0, 1], [-speed * 400, speed * 400]);

  return (
    <motion.div
      ref={ref}
      className={className}
      style={reduced ? undefined : { y }}
    >
      {children}
    </motion.div>
  );
};
