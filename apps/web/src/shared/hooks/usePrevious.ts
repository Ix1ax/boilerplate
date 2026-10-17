import { useState } from 'react';

/** Значение с предыдущего рендера (undefined на первом). */
export const usePrevious = <T>(value: T): T | undefined => {
  const [state, setState] = useState<{ value: T; prev: T | undefined }>({
    value,
    prev: undefined,
  });

  if (state.value !== value) {
    setState({ value, prev: state.value });
  }

  return state.prev;
};
