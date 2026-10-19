import { afterEach, describe, expect, it, vi } from 'vitest';

import { getSession } from './session';

afterEach(() => vi.unstubAllGlobals());

describe('проверка общей сессии администратора', () => {
  it('отправляет cookie в gateway без браузерного хранения токенов', async () => {
    const user = {
      id: '1',
      name: 'Администратор',
      email: 'admin@example.test',
      roles: ['admin'],
    };
    const fetch = vi.fn().mockResolvedValue(Response.json(user));
    vi.stubGlobal('fetch', fetch);
    const signal = new AbortController().signal;

    await expect(getSession(signal)).resolves.toEqual(user);
    expect(fetch).toHaveBeenCalledWith('/api/v1/me', {
      credentials: 'same-origin',
      cache: 'no-store',
      signal,
    });
  });

  it('считает гостем только ответ 401', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(null, { status: 401 })),
    );
    await expect(getSession(new AbortController().signal)).resolves.toBeNull();
  });

  it.each([403, 500, 503])(
    'не маскирует сбой %s под отсутствие сессии',
    async (status) => {
      vi.stubGlobal(
        'fetch',
        vi.fn().mockResolvedValue(new Response(null, { status })),
      );
      await expect(getSession(new AbortController().signal)).rejects.toThrow(
        'Не удалось проверить сессию',
      );
    },
  );

  it('сохраняет сетевую ошибку для повторной попытки', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockRejectedValue(new TypeError('Failed to fetch')),
    );
    await expect(getSession(new AbortController().signal)).rejects.toThrow();
  });
});
