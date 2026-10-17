'use client';

import { useCallback, useEffect, useRef } from 'react';

export const useDebounce = <Args extends unknown[]>(
  callback: (...args: Args) => void,
  delay: number,
) => {
  const callbackRef = useRef(callback);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(
    undefined,
  );

  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  const cancel = useCallback(() => {
    clearTimeout(timeoutRef.current);
    timeoutRef.current = undefined;
  }, []);

  useEffect(() => cancel, [cancel, delay]);

  const debounce = useCallback(
    (...args: Args) => {
      cancel();
      timeoutRef.current = setTimeout(() => {
        timeoutRef.current = undefined;
        callbackRef.current(...args);
      }, delay);
    },
    [cancel, delay],
  );

  return { debounce, cancel };
};
