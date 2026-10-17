'use client';

import {
  type ComponentProps,
  useCallback,
  useLayoutEffect,
  useRef,
} from 'react';

import { cn } from './utils';

interface TextareaProps extends ComponentProps<'textarea'> {
  /** Высота растёт вместе с текстом; `rows` задаёт минимальную высоту. */
  autoResize?: boolean;
}

const Textarea = ({
  className,
  autoResize = true,
  ref,
  onInput,
  ...props
}: TextareaProps) => {
  const node = useRef<HTMLTextAreaElement | null>(null);

  const fit = useCallback(() => {
    const element = node.current;
    if (!autoResize || !element) return;
    // Сначала сбрасываем высоту, иначе поле не уменьшается при удалении текста.
    element.style.height = 'auto';
    // Скрытое поле (вкладка, свёрнутый блок) не измерить — оставляем высоту по rows.
    if (!element.scrollHeight) return;
    const borders = element.offsetHeight - element.clientHeight;
    element.style.height = `${element.scrollHeight + borders}px`;
  }, [autoResize]);

  // После каждого рендера: значение могли задать программно (reset формы,
  // импорт черновика), а не вводом с клавиатуры.
  useLayoutEffect(fit);

  useLayoutEffect(() => {
    if (!autoResize) return;
    window.addEventListener('resize', fit);
    return () => window.removeEventListener('resize', fit);
  }, [autoResize, fit]);

  const setRef = useCallback(
    (element: HTMLTextAreaElement | null) => {
      node.current = element;
      if (typeof ref === 'function') ref(element);
      else if (ref) ref.current = element;
    },
    [ref],
  );

  return (
    <textarea
      data-slot="textarea"
      ref={setRef}
      onInput={(event) => {
        fit();
        onInput?.(event);
      }}
      className={cn(
        'flex min-h-20 w-full rounded-lg border border-input bg-background px-3 py-2.5 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:bg-input/50 disabled:text-muted-foreground aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 md:text-sm dark:disabled:bg-input/80 dark:aria-invalid:border-destructive/50 dark:aria-invalid:ring-destructive/40',
        autoResize && 'resize-none overflow-hidden',
        className,
      )}
      {...props}
    />
  );
};

export { Textarea };
