import { expect, test } from '@playwright/test';

const admin = {
  id: 'admin-review',
  email: 'admin@example.test',
  name: 'Администратор',
  roles: ['admin'],
};

test('адрес /admin сохраняет публичный origin и параметры', async ({
  page,
  baseURL,
}) => {
  await page.route('**/api/v1/me', (route) => route.fulfill({ status: 401 }));
  await page.goto('/admin?section=audit');
  const current = new URL(page.url());
  expect(current.origin).toBe(new URL(baseURL!).origin);
  expect(current.pathname).toMatch(/^\/admin\/?$/);
  expect(current.search).toBe('?section=audit');
});

test('гость входит с возвратом на исходный адрес admin', async ({ page }) => {
  await page.route('**/api/v1/me', (route) => route.fulfill({ status: 401 }));
  await page.goto('/admin/?section=audit');
  await expect(page).toHaveTitle('Панель администратора');
  await expect(
    page.getByRole('heading', { name: 'Вход в панель администратора' }),
  ).toBeVisible();
  await expect(
    page.getByRole('link', { name: 'Войти', exact: true }),
  ).toHaveAttribute('href', '/auth/login?next=%2Fadmin%2F%3Fsection%3Daudit');
});

test('обычный пользователь не видит инструменты администратора', async ({
  page,
}) => {
  await page.route('**/api/v1/me', (route) =>
    route.fulfill({ json: { ...admin, roles: ['user'] } }),
  );
  await page.goto('/admin/');
  await expect(
    page.getByRole('heading', { name: 'Доступ ограничен' }),
  ).toBeVisible();
  await expect(
    page.getByRole('link', { name: 'Открыть инструменты' }),
  ).toHaveCount(0);
});

test('администратор открывает отдельную SPA и может перейти в client', async ({
  page,
}) => {
  await page.route('**/api/v1/me', (route) => route.fulfill({ json: admin }));
  await page.goto('/admin/');
  await expect(
    page.getByRole('heading', { name: 'Панель администратора', exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole('link', { name: 'Открыть инструменты' }),
  ).toHaveAttribute('href', '/dashboard/admin');
  await page.getByRole('link', { name: 'Открыть сайт' }).click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page).toHaveTitle('Hackathon starter');
});

test('сбой сессии можно повторить без ложного выхода', async ({ page }) => {
  let failed = true;
  await page.route('**/api/v1/me', (route) =>
    route.fulfill(failed ? { status: 503 } : { json: admin }),
  );
  await page.goto('/admin/');
  await expect(
    page.getByRole('heading', { name: 'Не удалось проверить доступ' }),
  ).toBeVisible();
  failed = false;
  await page.getByRole('button', { name: 'Повторить' }).click();
  await expect(
    page.getByRole('heading', { name: 'Панель администратора', exact: true }),
  ).toBeVisible();
});

test('прямая ссылка и перезагрузка остаются в admin', async ({ page }) => {
  await page.route('**/api/v1/me', (route) => route.fulfill({ json: admin }));
  await page.goto('/admin/unknown');
  await expect(
    page.getByRole('heading', { name: 'Страница не найдена' }),
  ).toBeVisible();
  await page.reload();
  await expect(
    page.getByRole('heading', { name: 'Страница не найдена' }),
  ).toBeVisible();
  await page
    .getByRole('link', { name: 'Вернуться в панель администратора' })
    .click();
  await expect(
    page.getByRole('heading', { name: 'Панель администратора', exact: true }),
  ).toBeVisible();
});
