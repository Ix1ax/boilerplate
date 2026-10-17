import { useEffect, useRef } from 'react';

type EventTargetLike = Window | Document | HTMLElement;
type MaybeRef<T> = T | { current: T | null } | null;

const resolve = (target: MaybeRef<EventTargetLike>): EventTarget | null => {
  if (!target) return null;
  return 'current' in target ? target.current : target;
};

/**
 * Типобезопасный addEventListener с авто-очисткой. Handler можно передавать
 * инлайном — слушатель всегда вызывает актуальную версию.
 *
 * @example useEventListener('keydown', (e) => e.key === 'Escape' && close())
 * @example useEventListener('click', onClick, ref)
 */
export const useEventListener = <K extends keyof WindowEventMap>(
  type: K,
  handler: (event: WindowEventMap[K]) => void,
  target: MaybeRef<EventTargetLike> = window,
) => {
  const saved = useRef(handler);

  useEffect(() => {
    saved.current = handler;
  });

  useEffect(() => {
    const node = resolve(target);
    if (!node) return;

    const listener = (event: Event) =>
      saved.current(event as WindowEventMap[K]);
    node.addEventListener(type, listener);

    return () => node.removeEventListener(type, listener);
  }, [type, target]);
};
