import { useEffect, useState } from 'react';

/** Отложенное значение: обновляется через delay мс после последнего изменения. */
export const useDebounce = <T>(value: T, delay = 300): T => {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay);

    return () => clearTimeout(id);
  }, [value, delay]);

  return debounced;
};
