import { readFileSync } from 'node:fs';

import { expect, type Page } from '@playwright/test';

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
};

export const loginAsAdmin = (page: Page) =>
  login(page, env.ADMIN_DEMO_EMAIL, env.ADMIN_DEMO_PASSWORD);
