import type { RefObject } from 'react';

import { useEventListener } from './useEventListener';

/** Вызывает handler при клике/тапе вне элемента. Для поповеров, меню, модалок. */
export const useOnClickOutside = <T extends HTMLElement>(
  ref: RefObject<T | null>,
  handler: (event: PointerEvent) => void,
) => {
  useEventListener('pointerdown', (event) => {
    const el = ref.current;
    if (!el || el.contains(event.target as Node)) return;
    handler(event);
  });
};
