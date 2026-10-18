import { expect, type Page, test } from '@playwright/test';

import { login, loginAsAdmin } from './helpers';

const expectNoHorizontalOverflow = async (page: Page) => {
  const overflow = await page.evaluate(
    () =>
      document.documentElement.scrollWidth >
      document.documentElement.clientWidth,
  );
  expect(overflow, 'страница скроллится по горизонтали').toBe(false);
};

test('лендинг на мобильной ширине без горизонтального скролла', async ({
  page,
}) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  await expectNoHorizontalOverflow(page);
});

test('список задач и визард на мобильной ширине', async ({ page }) => {
  await login(page);
  await expectNoHorizontalOverflow(page);

  await page.getByRole('button', { name: 'Новая задача' }).click();
  await expect(page.getByLabel('Название задачи')).toBeVisible();
  await expectNoHorizontalOverflow(page);
});

test('админский дашборд на мобильной ширине', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/dashboard/admin');
  await expect(page.getByText('Всего задач')).toBeVisible();
  await expectNoHorizontalOverflow(page);
});
