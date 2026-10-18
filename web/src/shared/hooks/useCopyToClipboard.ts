import { useCallback, useState } from 'react';

import { logger } from '@/shared/lib';

const log = logger.child('clipboard');

const RESET_DELAY = 2_000;

/** Копирование в буфер с флагом `copied`, который сам сбрасывается через resetDelay. */
export const useCopyToClipboard = (resetDelay = RESET_DELAY) => {
  const [copied, setCopied] = useState(false);

  const copy = useCallback(
    async (text: string): Promise<boolean> => {
      try {
        await navigator.clipboard.writeText(text);
        setCopied(true);
        setTimeout(() => setCopied(false), resetDelay);
        return true;
      } catch (error) {
        log.warn('Не удалось скопировать в буфер обмена', error);
        setCopied(false);
        return false;
      }
    },
    [resetDelay],
  );

  return { copied, copy };
};
