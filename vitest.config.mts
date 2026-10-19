import path from 'node:path';
import { defineConfig } from 'vitest/config';
export default defineConfig({
  resolve: {
    alias: { '@': path.resolve(import.meta.dirname, 'frontend/client/src') },
  },
  test: {
    environment: 'node',
    include: [
      'frontend/client/src/**/*.test.ts',
      'frontend/admin/src/**/*.test.ts',
      'packages/**/*.test.ts',
    ],
  },
});
