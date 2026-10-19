import { readFileSync } from 'node:fs';

import { test as base, expect, type Page } from '@playwright/test';

// Демо-учётные данные создаёт make setup; в репозиторий они не попадают.
export const env = Object.fromEntries(
  readFileSync(new URL('../../.env', import.meta.url), 'utf8')
    .split('\n')
    .filter((line) => /^[A-Z][A-Z0-9_]*=/.test(line))
    .map((line) => [
      line.slice(0, line.indexOf('=')),
      line.slice(line.indexOf('=') + 1),
    ]),
);

export const login = async (
  page: Page,
  email = env.DEMO_EMAIL,
  password = env.DEMO_PASSWORD,
) => {
  await page.goto('/auth/login?next=/dashboard/tasks');
  await page.locator('#username').fill(email);
  await page.locator('#password').fill(password);
  await page.locator('#kc-login').click();
  await expect(page).toHaveURL(/\/dashboard\/tasks/);
  await appReady(page);
};

// Приложение сначала показывает вступление без разметки страницы: ждём основной контент.
export const appReady = (page: Page) =>
  expect(page.locator('main')).toBeVisible({ timeout: 15_000 });

export const loginAsAdmin = (page: Page) =>
  login(page, env.ADMIN_DEMO_EMAIL, env.ADMIN_DEMO_PASSWORD);

// Обучение (driver.js) запускается при первом входе и перекрывает экран: тесты сценариев
// проверяют сами экраны, поэтому для всех вкладок контекста обучение считается пройденным.
// Сам тур проверяется отдельно, где нужно, через обычный test из @playwright/test.
export const test = base.extend({
  context: async ({ context }, use) => {
    await context.addInitScript(() => {
      const getItem = Storage.prototype.getItem;
      Storage.prototype.getItem = function (key: string) {
        return key.startsWith('onboarding:')
          ? 'dismissed'
          : getItem.call(this, key);
      };
    });
    await use(context);
  },
});
