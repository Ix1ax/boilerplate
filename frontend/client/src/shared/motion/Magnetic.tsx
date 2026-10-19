import { type ReactNode, useEffect, useRef } from 'react';
import { cn } from 'cn';
import {
  motion,
  useMotionValue,
  useReducedMotion,
  useSpring,
} from 'motion/react';

interface MagneticProps {
  children: ReactNode;
  className?: string;
  /** Радиус притяжения вокруг элемента, px. */
  radius?: number;
  /** Доля смещения курсора, на которую тянется кнопка. */
  strength?: number;
}

/** Кнопка тянется к курсору в радиусе 80 px. Только для мыши (pointer: fine). */
export const Magnetic = ({
  children,
  className,
  radius = 80,
  strength = 0.3,
}: MagneticProps) => {
  const ref = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotion();
  const x = useMotionValue(0);
  const y = useMotionValue(0);
  const sx = useSpring(x, { stiffness: 260, damping: 18, mass: 0.6 });
  const sy = useSpring(y, { stiffness: 260, damping: 18, mass: 0.6 });

  useEffect(() => {
    if (reduced || !window.matchMedia('(pointer: fine)').matches) return;

    const move = (event: PointerEvent) => {
      const el = ref.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const dx = event.clientX - (rect.left + rect.width / 2);
      const dy = event.clientY - (rect.top + rect.height / 2);
      const near =
        Math.abs(dx) < rect.width / 2 + radius &&
        Math.abs(dy) < rect.height / 2 + radius;
      x.set(near ? dx * strength : 0);
      y.set(near ? dy * strength : 0);
    };
    const leave = () => {
      x.set(0);
      y.set(0);
    };

    window.addEventListener('pointermove', move);
    document.addEventListener('pointerleave', leave);
    return () => {
      window.removeEventListener('pointermove', move);
      document.removeEventListener('pointerleave', leave);
    };
  }, [reduced, radius, strength, x, y]);

  return (
    <motion.div
      ref={ref}
      className={cn('inline-flex', className)}
      style={{ x: sx, y: sy }}
    >
      {children}
    </motion.div>
  );
};
