import type { User } from '@starter/contracts';

export const getSession = async (signal: AbortSignal): Promise<User | null> => {
  const response = await fetch('/api/v1/me', {
    credentials: 'same-origin',
    cache: 'no-store',
    signal,
  });
  if (response.status === 401) return null;
  if (!response.ok)
    throw new Error('Не удалось проверить сессию. Попробуйте ещё раз.');
  return response.json() as Promise<User>;
};
