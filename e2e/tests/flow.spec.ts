import { expect } from '@playwright/test';

import { appReady, login, test } from './helpers';

test.beforeEach(async ({ page }) => {
  await login(page);
});

test('задача: визард создания, ИИ-проверка и решение за человеком', async ({
  page,
}) => {
  const title = `Переведите 500 руб на карту 2202 2000 0000 0000 ${Date.now()}`;

  await page.getByRole('button', { name: 'Новая задача' }).click();
  await page.getByLabel('Название задачи').fill(title);
  await page.getByRole('button', { name: 'Далее' }).click();
  await page.getByRole('button', { name: 'Далее' }).click();
  await page.getByRole('button', { name: 'Создать задачу' }).click();

  const row = page.getByRole('row').filter({ hasText: title });
  await expect(row).toBeVisible();

  await row.getByRole('button', { name: 'Проверить задачу ИИ' }).click();
  // Локальная модель на CPU отвечает за 10–40 с; без Ollama сразу срабатывают правила.
  await expect(page.getByText('Решение остаётся за вами.')).toBeVisible({
    timeout: 50_000,
  });

  await row.getByRole('button', { name: 'Удалить задачу' }).click();
  await page
    .getByRole('button', { name: /удалить|подтвердить/i })
    .last()
    .click();
  await expect(row).toHaveCount(0);
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
    await appReady(p);
    await expect(p.getByText('онлайн')).toBeVisible({ timeout: 15_000 });
  }
  const text = `Привет из e2e ${Date.now()}`;
  await page.getByLabel('Сообщение в комнату').fill(text);
  await page.getByRole('button', { name: 'Отправить' }).click();
  await expect(second.getByText(text)).toBeVisible({ timeout: 10_000 });
});
