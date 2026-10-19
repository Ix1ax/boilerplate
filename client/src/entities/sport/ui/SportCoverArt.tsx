import type { ReactNode } from 'react';
import { cn } from 'cn';

import { getSportVisual } from '../lib/sportVisual';
import { SportGlyph } from './SportGlyph';

interface SportCoverArtProps {
  sport: string;
  className?: string;
  /** Диаметры колец вокруг значка. */
  rings?: number[];
  glyphSize?: number;
  children?: ReactNode;
}

/** Радар-обложка встречи: кольца расходятся от предмета спорта. */
export const SportCoverArt = ({
  sport,
  className,
  rings = [96, 150, 212],
  glyphSize = 64,
  children,
}: SportCoverArtProps) => {
  const { icon, cover } = getSportVisual(sport);
  return (
    <div
      className={cn(
        'relative grid place-items-center overflow-hidden',
        className,
      )}
      style={{ background: cover.bg }}
    >
      {rings.map((size) => (
        <span
          key={size}
          aria-hidden="true"
          className="absolute top-1/2 left-1/2 -translate-1/2 rounded-full border-[1.5px]"
          style={{ width: size, height: size, borderColor: cover.ring }}
        />
      ))}
      <SportGlyph
        icon={icon}
        color={cover.ink}
        size={glyphSize}
        className="relative"
      />
      {children}
    </div>
  );
};
