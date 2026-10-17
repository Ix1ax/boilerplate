import { useEffect, useRef } from 'react';

/** Декларативный setInterval. delay=null ставит на паузу. */
export const useInterval = (callback: () => void, delay: number | null) => {
  const saved = useRef(callback);

  useEffect(() => {
    saved.current = callback;
  });

  useEffect(() => {
    if (delay === null) return;
    const id = setInterval(() => saved.current(), delay);

    return () => clearInterval(id);
  }, [delay]);
};
