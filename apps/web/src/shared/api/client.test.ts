import { afterEach, expect, it, vi } from 'vitest';
import { api, ApiError } from './client';
afterEach(() => vi.unstubAllGlobals());
it('сохраняет код ошибки API и идентификатор запроса', async () => {
  vi.stubGlobal(
    'fetch',
    vi
      .fn()
      .mockResolvedValue(
        Response.json(
          { error: { code: 'rate_limit', message: 'Slow down' } },
          { status: 429, headers: { 'x-request-id': 'r-1' } },
        ),
      ),
  );
  await expect(api('/tasks')).rejects.toMatchObject({
    status: 429,
    code: 'rate_limit',
    message: 'Slow down',
    requestId: 'r-1',
  });
});
it('сохраняет HTTP-статус, если upstream вернул ответ без JSON', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(new Response('Gateway failed', { status: 502 })),
  );
  await expect(api('/tasks')).rejects.toBeInstanceOf(ApiError);
});
it('не подставляет JSON content type при загрузке файла', async () => {
  const fetch = vi.fn().mockResolvedValue(Response.json({ id: 'file' }));
  vi.stubGlobal('fetch', fetch);
  const data = new FormData();
  data.append('file', new Blob(['test']), 'test.txt');
  await api('/files', { method: 'POST', body: data });
  expect(fetch.mock.calls[0][1].headers).toEqual({});
});
it('обрабатывает пустой ответ при удалении', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
  );
  await expect(api('/tasks/1', { method: 'DELETE' })).resolves.toBeUndefined();
});
