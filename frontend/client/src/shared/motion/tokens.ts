// Единый источник правды для анимаций — не хардкодь длительности/кривые по месту.
// Значения из моушн-системы «Вместе»: в приложении ≤ 280 мс на любое действие пользователя,
// «кинематографичные» 900 мс — только на лендинге.

/** Длительности в секундах (формат motion). */
export const DURATION = {
  /** Фокус, нажатие, переключатель. */
  instant: 0.1,
  /** Ховер, цвет, чипы фильтров. */
  quick: 0.18,
  /** Смена статуса, тост, модалка. */
  base: 0.28,
  /** Карточки, ревил. */
  slow: 0.48,
  /** Только лендинг: заголовки, переходы секций. */
  cine: 0.9,
  /** @deprecated Используйте `quick`. */
  fast: 0.18,
} as const;

type Bezier = [number, number, number, number];

/** Easing-кривые (cubic-bezier [x1, y1, x2, y2]). */
export const EASING = {
  /** --e-out · expo: появление, ревилы, нижний лист. */
  out: [0.16, 1, 0.3, 1] as Bezier,
  /** --e-inout: смена слов, переходы между экранами. */
  inOut: [0.65, 0, 0.35, 1] as Bezier,
  /** --e-spring: кнопки, «Вы записаны». */
  spring: [0.34, 1.56, 0.64, 1] as Bezier,
  standard: [0.2, 0, 0, 1] as Bezier,
  emphasized: [0.3, 0, 0, 1] as Bezier,
};

/** Шаг между элементами списка; максимум 8 элементов в каскаде. */
export const STAGGER = { step: 0.06, max: 8 } as const;

/** Задержка i-го элемента каскада. */
export const staggerDelay = (index: number) =>
  Math.min(index, STAGGER.max - 1) * STAGGER.step;

/** Пружина для «живых» переходов (drag, layout). */
export const SPRING = {
  type: 'spring',
  stiffness: 400,
  damping: 30,
} as const;

/** Мягкая пружина с отскоком — отклик кнопок и значков статуса. */
export const SPRING_POP = {
  type: 'spring',
  stiffness: 520,
  damping: 18,
  mass: 0.8,
} as const;
