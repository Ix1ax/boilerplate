import path from 'node:path';
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  testMatch: 'microfrontends.spec.ts',
  workers: 1,
  timeout: 30_000,
  reporter: 'list',
  use: { baseURL: 'http://localhost:5180', browserName: 'chromium' },
  webServer: {
    command: 'npm run dev',
    cwd: path.resolve(import.meta.dirname, '..'),
    url: 'http://localhost:5180/admin/',
    env: { CLIENT_DEV_PORT: '5180', ADMIN_DEV_PORT: '5181' },
    timeout: 60_000,
  },
});
