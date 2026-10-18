// Единый источник правды для анимаций — не хардкодь длительности/кривые по месту.

/** Длительности в секундах (формат motion). */
export const DURATION = {
  fast: 0.15,
  base: 0.25,
  slow: 0.4,
} as const;

/** Easing-кривые (cubic-bezier [x1, y1, x2, y2]). */
export const EASING = {
  standard: [0.2, 0, 0, 1] as [number, number, number, number],
  emphasized: [0.3, 0, 0, 1] as [number, number, number, number],
};

/** Пружина для «живых» переходов (drag, layout). */
export const SPRING = {
  type: 'spring',
  stiffness: 400,
  damping: 30,
} as const;
