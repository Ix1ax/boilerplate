import { expect, test } from '@playwright/test';

// Проверка настоящей cookie-сессии gateway, без имитации API и сохранения токенов.
test('локальная регистрация и ошибка пароля сохраняют email в форме', async ({
  page,
  request,
}) => {
  const providers = await request.get('/auth/providers');
  test.skip(
    !(await providers.json()).local?.enabled,
    'Локальные аккаунты отключены',
  );
  const email = `review-${Date.now()}@example.test`;
  const password = `Review-${Date.now()}-Aa!`;
  await page.goto('/login');
  await page.getByRole('tab', { name: 'Регистрация', exact: true }).click();
  await page.getByLabel('Имя', { exact: true }).fill('Проверка входа');
  await page.getByLabel('Email', { exact: true }).fill(email);
  await page.getByLabel('Пароль', { exact: true }).fill(password);
  await page
    .getByRole('button', { name: 'Создать аккаунт', exact: true })
    .click();
  await expect(page).toHaveURL(/\/events$/);
  const me = await page.request.get('/api/v1/me');
  expect(me.ok()).toBeTruthy();
  const loggedOut = await page.request.post('/auth/logout', {
    headers: { Origin: new URL(page.url()).origin },
  });
  expect(loggedOut.ok()).toBeTruthy();
  await page.goto('/login');
  await page.getByLabel('Email', { exact: true }).fill(email);
  await page.getByLabel('Пароль', { exact: true }).fill('wrong-password');
  await page.getByRole('button', { name: 'Войти', exact: true }).click();
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByLabel('Email', { exact: true })).toHaveValue(email);
  await expect(page.getByLabel('Пароль', { exact: true })).toHaveValue(
    'wrong-password',
  );
  await page.getByLabel('Пароль', { exact: true }).fill(password);
  await page.getByRole('button', { name: 'Войти', exact: true }).click();
  await expect(page).toHaveURL(/\/events$/);
});
