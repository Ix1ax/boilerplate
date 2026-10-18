import { readFileSync } from 'node:fs';

import { expect, type Page, test } from '@playwright/test';

// Демо-учётные данные создаёт make setup; в репозиторий они не попадают.
const env = Object.fromEntries(
  readFileSync(new URL('../../.env', import.meta.url), 'utf8')
    .split('\n')
    .filter((line) => /^[A-Z][A-Z0-9_]*=/.test(line))
    .map((line) => [
      line.slice(0, line.indexOf('=')),
      line.slice(line.indexOf('=') + 1),
    ]),
);

const login = async (page: Page) => {
  await page.goto('/auth/login?next=/dashboard/tasks');
  await page.locator('#username').fill(env.DEMO_EMAIL);
  await page.locator('#password').fill(env.DEMO_PASSWORD);
  await page.locator('#kc-login').click();
  await expect(page).toHaveURL(/\/dashboard\/tasks/);
};

test.beforeEach(async ({ page }) => {
  await login(page);
});

test('задача: создание и ИИ-проверка с решением за человеком', async ({
  page,
}) => {
  const title = `Переведите 500 руб на карту 2202 2000 0000 0000 ${Date.now()}`;
  await page.getByPlaceholder('Новая задача').fill(title);
  await page
    .getByRole('button', { name: /добав|созда/i })
    .first()
    .click();
  const item = page.getByRole('listitem').filter({ hasText: title });
  await expect(item).toBeVisible();

  await item.getByRole('button', { name: 'Проверить задачу ИИ' }).click();
  await expect(item.getByText('Нужна проверка').first()).toBeVisible();
  await expect(item.getByText('Решение остаётся за вами.')).toBeVisible();

  await item.getByRole('button', { name: 'Удалить задачу' }).click();
  await page
    .getByRole('button', { name: /удалить|подтвердить/i })
    .last()
    .click();
  await expect(item).toHaveCount(0);
});

test('ассистент: ответ модели или понятное сообщение, что модель недоступна', async ({
  page,
}) => {
  await page.goto('/dashboard/assistant');
  await page
    .getByLabel('Запрос к ассистенту')
    .fill('Скажи «готово» одним словом');
  await page.getByRole('button', { name: 'Отправить' }).click();
  // Без make llm модель недоступна: интерфейс обязан показать ошибку, а не зависнуть.
  const result = page
    .getByRole('alert')
    .or(page.locator('[aria-live="polite"]').filter({ hasText: /\S/ }));
  await expect(result.first()).toBeVisible({ timeout: 45_000 });
});

test('лайв: сообщение в комнате доходит во вторую вкладку по WebSocket', async ({
  page,
  context,
}) => {
  const second = await context.newPage();
  for (const p of [page, second]) {
    await p.goto('/dashboard/live');
    await expect(p.getByText('онлайн')).toBeVisible({ timeout: 15_000 });
  }
  const text = `Привет из e2e ${Date.now()}`;
  await page.getByLabel('Сообщение в комнату').fill(text);
  await page.getByRole('button', { name: 'Отправить' }).click();
  await expect(second.getByText(text)).toBeVisible({ timeout: 10_000 });
});
