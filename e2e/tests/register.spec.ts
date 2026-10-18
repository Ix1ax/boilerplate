import { expect, test } from '@playwright/test';

// Регистрация с подтверждением почты: Keycloak шлёт письмо в Mailpit, ссылка из письма
// подтверждает адрес и пускает в приложение. Демо-пользователи из make setup уже подтверждены.
const mailpit = process.env.E2E_MAILPIT_URL ?? 'http://localhost:8025';

test('регистрация: письмо в Mailpit и подтверждение почты', async ({
  page,
  request,
}) => {
  const email = `e2e-${Date.now()}@example.test`;
  const password = `E2e-pass-${Date.now()}`;

  await page.goto('/auth/login?next=/dashboard/tasks');
  await page.locator('#kc-registration a').click();
  // При включённом подтверждении почты Keycloak 26 не спрашивает пароль в форме регистрации:
  // его задают после перехода по ссылке из письма.
  await page.locator('#email').fill(email);
  await page.locator('#firstName').fill('Тест');
  await page.locator('#lastName').fill('Регистрация');
  await page.locator('input[type=submit]').click();

  // Без подтверждения в приложение не пускает — Keycloak просит проверить почту.
  await expect(page.locator('#kc-page-title')).toContainText(/email/i);

  // Письмо приходит не мгновенно — опрашиваем API Mailpit.
  let id = '';
  await expect
    .poll(
      async () => {
        const res = await request.get(`${mailpit}/api/v1/search`, {
          params: { query: `to:"${email}"` },
        });
        id = (await res.json()).messages?.[0]?.ID ?? '';
        return id;
      },
      { timeout: 20_000 },
    )
    .not.toBe('');

  const message = await (
    await request.get(`${mailpit}/api/v1/message/${id}`)
  ).json();
  const link = String(message.Text).match(/https?:\/\/\S+action-token\S+/)?.[0];
  expect(link, 'ссылка подтверждения в письме').toBeTruthy();

  // Ссылка открывается в том же браузере: Keycloak подтверждает адрес, просит задать пароль
  // и возвращает в приложение уже залогиненным.
  await page.goto(link!);
  await page.locator('#password-new').fill(password);
  await page.locator('#password-confirm').fill(password);
  await page.getByRole('button', { name: /submit|отправить/i }).click();
  await expect(page).toHaveURL(/\/dashboard\/tasks/);
});
