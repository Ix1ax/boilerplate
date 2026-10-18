import type { ReactNode } from 'react';
import { useLayoutEffect, useMemo, useRef, useState } from 'react';
import { cn } from 'cn';
import { AnimatePresence, motion } from 'motion/react';

export type AnimatedTextLinesSegment = {
  className?: string;
  text: string;
};

export type AnimatedTextLinesLayer<State extends string = string> = {
  id: string;
  segments: readonly AnimatedTextLinesSegment[];
  state: State;
};

export type AnimatedTextLinesSlot<State extends string = string> = {
  id: string;
  layers: readonly AnimatedTextLinesLayer<State>[];
  sizerText: string;
  sizerTexts?: readonly string[];
};

export interface AnimatedTextLinesProps<State extends string = string> {
  className?: string;
  lineClassName?: string;
  layerClassName?: string;
  slots: readonly AnimatedTextLinesSlot<State>[];
  disabled?: boolean;
  duration?: number;
  autoWrap?: boolean;
  autoWrapWidthBuffer?: number;
}

const DEFAULT_AUTO_WRAP_WIDTH_BUFFER = 1;

const splitWords = (line: string) => line.trim().split(/\s+/u).filter(Boolean);

const createMeasurer = (element: HTMLElement) => {
  const canvas = document.createElement('canvas');
  const context = canvas.getContext('2d');
  const style = window.getComputedStyle(element);

  if (!context) {
    const fontSize = Number.parseFloat(style.fontSize) || 14;
    return (text: string) => text.length * fontSize * 0.6;
  }

  context.font =
    style.font ||
    [
      style.fontStyle,
      style.fontVariant,
      style.fontWeight,
      style.fontSize,
      style.fontFamily,
    ].join(' ');

  return (text: string) => context.measureText(text).width;
};

const wrapLine = (
  line: string,
  maxWidth: number,
  measure: (text: string) => number,
) => {
  const words = splitWords(line);

  if (words.length === 0 || maxWidth <= 0) {
    return line ? [line] : [''];
  }

  const lines: string[] = [];
  let current = '';

  for (const word of words) {
    const next = current ? `${current} ${word}` : word;

    if (measure(next) <= maxWidth) {
      current = next;
      continue;
    }

    if (current) {
      lines.push(current);
    }
    current = word;
  }

  if (current) {
    lines.push(current);
  }

  return lines.length > 0 ? lines : [line];
};

const layerText = (layer: AnimatedTextLinesLayer) =>
  layer.segments.map((segment) => segment.text).join('');

const wrapSlot = <State extends string>(
  slot: AnimatedTextLinesSlot<State>,
  maxWidth: number,
  measure: (text: string) => number,
): AnimatedTextLinesSlot<State>[] => {
  const wrapped = slot.layers.map((layer) => ({
    layer,
    lines: wrapLine(layerText(layer), maxWidth, measure),
  }));
  const lineCount = Math.max(1, ...wrapped.map(({ lines }) => lines.length));

  return Array.from({ length: lineCount }, (_, index) => {
    const text = wrapped[0]?.lines[index] ?? ' ';

    return {
      id: `${slot.id}-${index}`,
      sizerText: text,
      layers: wrapped.flatMap(({ layer, lines }) => {
        const line = lines[index];
        if (typeof line !== 'string') return [];

        return [
          {
            ...layer,
            id: `${layer.id}-${index}`,
            segments:
              layer.segments.length === 1
                ? [{ ...layer.segments[0], text: line || ' ' }]
                : [{ text: line || ' ' }],
          },
        ];
      }),
    };
  });
};

const useWrappedSlots = <State extends string>(
  slots: readonly AnimatedTextLinesSlot<State>[],
  autoWrap: boolean,
  widthBuffer: number,
) => {
  const rootRef = useRef<HTMLDivElement>(null);
  const [wrapped, setWrapped] = useState<
    readonly AnimatedTextLinesSlot<State>[] | null
  >(null);

  useLayoutEffect(() => {
    const root = rootRef.current;
    if (!root || !autoWrap) {
      setWrapped(null);
      return;
    }

    const measureElement = root.parentElement ?? root;

    const recalculate = () => {
      const maxWidth = measureElement.clientWidth - widthBuffer;
      if (maxWidth <= 0) return;

      setWrapped(
        slots.flatMap((slot) =>
          wrapSlot(slot, maxWidth, createMeasurer(measureElement)),
        ),
      );
    };

    recalculate();

    const observer = new ResizeObserver(recalculate);
    observer.observe(measureElement);

    return () => observer.disconnect();
  }, [autoWrap, widthBuffer, slots]);

  return { rootRef, slots: autoWrap ? (wrapped ?? slots) : slots };
};

interface AnimatedLayerProps<State extends string = string> {
  layer: AnimatedTextLinesLayer<State>;
  duration: number;
  disabled: boolean;
  layerClassName?: string;
}

const AnimatedLayer = <State extends string>({
  layer,
  duration,
  disabled,
  layerClassName,
}: AnimatedLayerProps<State>) => {
  const children: ReactNode = layer.segments.map((segment, index) => (
    <span className={segment.className} key={`${layer.id}-${index}`}>
      {segment.text}
    </span>
  ));

  const offsetByState: Record<string, string> = {
    before: '-100%',
    after: '100%',
  };
  const offset = offsetByState[layer.state] ?? '0%';

  return (
    <motion.span
      aria-hidden
      className={cn('absolute inset-0 block min-w-full', layerClassName)}
      animate={{ y: offset }}
      transition={
        disabled
          ? { duration: 0 }
          : { duration: duration / 1000, ease: [0.4, 0, 0.2, 1] }
      }
    >
      {children}
    </motion.span>
  );
};

/**
 * Построчная текстовая анимация: несколько слоёв одной строки
 * переключаются между состояниями ("before" / текущее / "after"),
 * уезжая по вертикали. Ширина строки держится «сайзером» —
 * невидимым текстом, чтобы layout не прыгал при переключении слоёв.
 */
export const AnimatedTextLines = <State extends string = string>({
  className,
  lineClassName,
  layerClassName,
  slots,
  disabled = false,
  duration = 700,
  autoWrap = false,
  autoWrapWidthBuffer = DEFAULT_AUTO_WRAP_WIDTH_BUFFER,
}: AnimatedTextLinesProps<State>) => {
  const { rootRef, slots: renderedSlots } = useWrappedSlots(
    slots,
    autoWrap,
    autoWrapWidthBuffer,
  );
  const key = useMemo(
    () => renderedSlots.map((slot) => slot.id).join('|'),
    [renderedSlots],
  );

  return (
    <div className={cn('contents', className)} key={key} ref={rootRef}>
      {renderedSlots.map((slot) => {
        const sizerTexts = slot.sizerTexts ?? [slot.sizerText];

        return (
          <div
            className={cn(
              'relative block w-max min-w-full overflow-hidden',
              lineClassName,
            )}
            key={slot.id}
          >
            <div className="grid w-max min-w-full invisible">
              {sizerTexts.map((text) => (
                <span
                  className="col-start-1 row-start-1 block min-w-full"
                  key={text}
                >
                  {text || ' '}
                </span>
              ))}
            </div>

            <AnimatePresence initial={false}>
              {slot.layers.map((layer) => (
                <AnimatedLayer
                  disabled={disabled}
                  duration={duration}
                  key={layer.id}
                  layer={layer}
                  layerClassName={layerClassName}
                />
              ))}
            </AnimatePresence>
          </div>
        );
      })}
    </div>
  );
};
