import type { CSSProperties } from 'react';
import { cn } from 'cn';

type Tone = 'ink' | 'gold' | 'blue' | 'white';

const TONE_RGB: Record<Tone, string> = {
  ink: '18 18 18',
  gold: '250 213 83',
  blue: '35 64 216',
  white: '255 255 255',
};

interface RadarRingsProps {
  /** Диаметры колец в px — от внутреннего к внешнему. */
  sizes: number[];
  tone?: Tone;
  /** Прозрачность внутреннего кольца; наружу убывает. */
  strength?: number;
  /** Положение центра колец: классы top/left/right/bottom. */
  className?: string;
  style?: CSSProperties;
}

/**
 * Мотив «волны радара»: 3–4 кольца расходятся от центра, прозрачность убывает наружу.
 * Только декор: без бесконечной анимации и скрыт от скринридеров.
 */
export const RadarRings = ({
  sizes,
  tone = 'gold',
  strength = 0.6,
  className,
  style,
}: RadarRingsProps) => {
  const rgb = TONE_RGB[tone];
  return (
    <span
      aria-hidden="true"
      className={cn('pointer-events-none absolute size-0', className)}
      style={style}
    >
      {sizes.map((size, index) => {
        const alpha = strength * (1 - (index / sizes.length) * 0.7);
        return (
          <span
            key={size}
            className="absolute -translate-1/2 rounded-full border-[1.5px]"
            style={{
              width: size,
              height: size,
              borderColor: `rgb(${rgb} / ${alpha.toFixed(2)})`,
              borderWidth: index === 0 && strength >= 0.5 ? 2 : undefined,
            }}
          />
        );
      })}
    </span>
  );
};
