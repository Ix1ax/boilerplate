import { afterEach, expect, it, vi } from 'vitest';

afterEach(() => {
  vi.unstubAllEnvs();
  vi.resetModules();
});
it('использует same-origin API в production-сборке без локального env', async () => {
  vi.stubEnv('VITE_API_URL', undefined);
  vi.resetModules();
  const { env } = await import('./env');
  expect(env.VITE_API_URL).toBe('/api/v1');
});
