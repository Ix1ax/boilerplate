import { expect, test } from '@playwright/test';

// Обычная регистрация по почте и паролю (сервис identity). Запросы идут из контекста браузера,
// поэтому cookie сессии ставится этому же браузеру — как после отправки формы на экране входа.
test('регистрация по почте: сразу вход, профиль и выход', async ({ page }) => {
  const email = `e2e-${Date.now()}@example.test`;
  const password = `E2epass${Date.now()}`;
  await page.goto('/');
  const origin = new URL(page.url()).origin;
  const post = (path: string, data: object) =>
    page.request.post(path, { data, headers: { Origin: origin } });

  const providers = await (await page.request.get('/auth/providers')).json();
  expect(providers.local.enabled).toBe(true);

  const registered = await post('/auth/local/register', {
    email,
    password,
    name: 'Тест Регистрация',
  });
  expect(registered.status()).toBe(201);
  const me = await (await page.request.get('/api/v1/me')).json();
  expect(me).toMatchObject({ email, provider: 'local', roles: ['user'] });

  expect(
    (
      await post('/auth/local/register', { email, password, name: 'Дубль' })
    ).status(),
  ).toBe(409);
  const logout = await post('/auth/logout', {});
  expect((await logout.json()).url).toBe(`${origin}/`);
  expect((await page.request.get('/api/v1/me')).status()).toBe(401);
  expect((await post('/auth/local/login', { email, password })).status()).toBe(
    200,
  );
});

// «Столото ID» — единый вход компании: учётные записи там не заводят сами пользователи.
test('в «Столото ID» нет самостоятельной регистрации', async ({ page }) => {
  await page.goto('/auth/login?next=/dashboard/tasks');
  await expect(page.locator('#kc-login')).toBeVisible();
  await expect(page.locator('#kc-registration')).toHaveCount(0);
});
