/** Промис-пауза. @example await sleep(300) */
export const sleep = (ms: number) =>
  new Promise<void>((resolve) => setTimeout(resolve, ms));

/** Отличает отмену запроса (AbortController / axios cancel) от настоящей ошибки. */
export const isAbortError = (error: unknown): boolean => {
  if (error instanceof DOMException && error.name === 'AbortError') return true;
  if (
    error instanceof Error &&
    (error.name === 'AbortError' || error.name === 'CanceledError')
  ) {
    return true;
  }
  return false;
};
