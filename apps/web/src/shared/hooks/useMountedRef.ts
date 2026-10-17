import { useEffect, useRef } from 'react';

/**
 * Ref со статусом монтирования. Гвард для ручных async, чтобы не ставить стейт
 * после размонтирования: `if (!mounted.current) return`.
 */
export const useMountedRef = () => {
  const mounted = useRef(true);

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  return mounted;
};
