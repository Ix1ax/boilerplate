import { useEffect, useRef } from 'react';

/** Декларативный setTimeout. delay=null отменяет. Перезапускается при смене delay. */
export const useTimeout = (callback: () => void, delay: number | null) => {
  const saved = useRef(callback);

  useEffect(() => {
    saved.current = callback;
  });

  useEffect(() => {
    if (delay === null) return;
    const id = setTimeout(() => saved.current(), delay);
    return () => clearTimeout(id);
  }, [delay]);
};
