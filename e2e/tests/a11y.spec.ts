import AxeBuilder from '@axe-core/playwright';
import { expect, type Page } from '@playwright/test';

import { appReady, login, loginAsAdmin, test } from './helpers';

// Виджет widgets/accessibility — сторонний компонент (open-accessibility-widget),
// его разметку не проверяем: нарушения внутри него не относятся к нашему коду.
const AXE_EXCLUDE = '[class*="acsb"]';

// axe меряет контраст по текущему кадру: посреди CSS-перехода (например, смены активной вкладки)
// цвета промежуточные, и проверка падает случайно. Ждём, пока доиграют все конечные анимации;
// бесконечные (декоративный радар) не ждём — они не меняют цвета текста.
const animationsSettled = (page: Page) =>
  page.waitForFunction(
    () =>
      document
        .getAnimations()
        .every(
          (a) =>
            a.playState !== 'running' ||
            a.effect?.getTiming().iterations === Infinity,
        ),
    undefined,
    { timeout: 10_000 },
  );

const checkPage = async (page: Page) => {
  await appReady(page);
  await animationsSettled(page);
  const results = await new AxeBuilder({ page }).exclude(AXE_EXCLUDE).analyze();
  expect(
    results.violations,
    JSON.stringify(results.violations, null, 2),
  ).toEqual([]);
};

test('лендинг доступен (axe)', async ({ page }) => {
  await page.goto('/');
  await checkPage(page);
});

test('дашборд задач и визард доступны (axe)', async ({ page }) => {
  await login(page);
  await checkPage(page);

  await page.getByRole('button', { name: 'Новая задача' }).click();
  await checkPage(page);
});

test('админский дашборд доступен (axe)', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/dashboard/admin');
  await checkPage(page);

  await page.getByRole('tab', { name: 'Журнал' }).click();
  await checkPage(page);
});
