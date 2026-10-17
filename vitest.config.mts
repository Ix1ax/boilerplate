import path from 'node:path';
import { defineConfig } from 'vitest/config';
export default defineConfig({
  resolve: {
    alias: { '@': path.resolve(import.meta.dirname, 'apps/web/src') },
  },
  test: {
    environment: 'node',
    include: ['apps/web/src/**/*.test.ts', 'packages/**/*.test.ts'],
  },
});
