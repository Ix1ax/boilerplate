import { useCallback, useState } from 'react';

export type UseToggle = {
  value: boolean;
  toggle: () => void;
  setTrue: () => void;
  setFalse: () => void;
  setValue: (next: boolean) => void;
};

/** Булев стейт с готовыми экшенами — для модалок, дропдаунов, аккордеонов. */
export const useToggle = (initial = false): UseToggle => {
  const [value, setValue] = useState(initial);

  const toggle = useCallback(() => setValue((prev) => !prev), []);
  const setTrue = useCallback(() => setValue(true), []);
  const setFalse = useCallback(() => setValue(false), []);

  return { value, toggle, setTrue, setFalse, setValue };
};
