import type { ComponentType, SVGProps } from 'react';
import {
  Bike,
  Flower2,
  Footprints,
  Mountain,
  Snowflake,
  Target,
  Volleyball,
  Waves,
} from 'lucide-react';

import type { SportIconKey } from '../lib/sportVisual';

// Значки из дизайн-системы: тонкая обводка 1,6 на сетке 24.
const PATHS: Partial<Record<SportIconKey, string>> = {
  chess: 'M7 20h10M8 17h8l-1-7H9zM8 10V5h2v2h1.5V5h1v2H14V5h2v5',
  gym: 'M6 8v8M18 8v8M3 10v4M21 10v4M6 12h12',
  football: 'M12 3a9 9 0 1 0 0 18a9 9 0 1 0 0-18M12 7l4 3-1.5 4.5h-5L8 10z',
  run: 'M14 3.5a1.8 1.8 0 1 0 0 .01M9 21l3-6 3 2v4M6 12l3-3 4 1 3 3M12 15l-2-3',
  medal:
    'M8 3h8l-2.5 6h-3zM12 9.5a5.5 5.5 0 1 0 0 11a5.5 5.5 0 1 0 0-11M12 12.6l1.1 2.2 2.4.3-1.7 1.7.4 2.4-2.2-1.2-2.2 1.2.4-2.4-1.7-1.7 2.4-.3z',
  petanque:
    'M3.5 15a3.5 3.5 0 1 0 7 0a3.5 3.5 0 1 0-7 0M12.5 11a4 4 0 1 0 8 0a4 4 0 1 0-8 0M10.8 20.2a1.2 1.2 0 1 0 2.4 0a1.2 1.2 0 1 0-2.4 0',
  tennis:
    'M12 3a9 9 0 1 0 0 18a9 9 0 1 0 0-18M5.6 5.6c3 3 3 9.8 0 12.8M18.4 5.6c-3 3-3 9.8 0 12.8',
};

const LUCIDE: Partial<
  Record<SportIconKey, ComponentType<SVGProps<SVGSVGElement>>>
> = {
  volleyball: Volleyball,
  bike: Bike,
  swim: Waves,
  climb: Mountain,
  walk: Footprints,
  ski: Snowflake,
  yoga: Flower2,
  other: Target,
};

interface SportGlyphProps {
  icon: SportIconKey;
  color: string;
  size?: number;
  className?: string;
}

/** Значок вида спорта для обложек и списков. Декоративный: смысл несёт подпись рядом. */
export const SportGlyph = ({
  icon,
  color,
  size = 64,
  className,
}: SportGlyphProps) => {
  const path = PATHS[icon];
  if (path) {
    return (
      <svg
        aria-hidden="true"
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke={color}
        strokeWidth={1.6}
        strokeLinecap="round"
        strokeLinejoin="round"
        className={className}
      >
        <path d={path} />
      </svg>
    );
  }
  const Icon = LUCIDE[icon] ?? Target;
  return (
    <Icon
      aria-hidden="true"
      width={size}
      height={size}
      stroke={color}
      strokeWidth={1.6}
      className={className}
    />
  );
};
