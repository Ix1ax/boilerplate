import { defineConfig, devices } from '@playwright/test';

// Запускается против живого стека (make up && make seed): make e2e.
export default defineConfig({
  testDir: './tests',
  timeout: 60_000,
  retries: 1,
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:3000',
    trace: 'retain-on-failure',
    // Без анимаций вступление приложения длится 200 мс вместо 4 с, а снимки и клики стабильны.
    reducedMotion: 'reduce',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
      testIgnore: /mobile\.spec\.ts/,
    },
    {
      // Эмуляция вьюпорта iPhone 13 на Chromium: make e2e ставит только него (webkit не скачан).
      name: 'mobile',
      use: { ...devices['iPhone 13'], browserName: 'chromium' },
      testMatch: /mobile\.spec\.ts/,
    },
  ],
});
