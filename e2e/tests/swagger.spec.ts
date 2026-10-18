import { expect, test } from '@playwright/test';

import { env } from './helpers';

// Вход в Swagger через Keycloak (клиент hackathon-swagger, PKCE) и запрос с полученным токеном.
// Отдельный клиент: вход в само приложение этим не затрагивается (его проверяет flow.spec.ts).
test('swagger: вход через Keycloak и запрос /api/v1/me', async ({
  page,
  context,
}) => {
  await page.goto('/api/docs?urls.primaryName=gateway');
  await page.locator('.auth-wrapper .authorize').click();

  // Слово «Keycloak» есть и в описаниях других схем — ищем контейнер по заголовку OAuth2-схемы.
  const keycloak = page
    .locator('.dialog-ux .auth-container')
    .filter({ has: page.locator('h4', { hasText: 'keycloak (OAuth2' }) });
  // client_id и области доступа заполнены через initOAuth в index.html.
  await expect(keycloak.locator('#client_id_authorizationCode')).toHaveValue(
    'hackathon-swagger',
  );

  const popupPromise = context.waitForEvent('page');
  await keycloak.locator('button.authorize').click();

  // Keycloak открывается во всплывающем окне и после входа возвращает на oauth2-redirect.html.
  const popup = await popupPromise;
  await popup.locator('#username').fill(env.DEMO_EMAIL);
  await popup.locator('#password').fill(env.DEMO_PASSWORD);
  // Окно закрывается сразу после входа — подписываемся на close до клика, иначе событие можно пропустить.
  const closed = popup.waitForEvent('close');
  await popup.locator('#kc-login').click({ noWaitAfter: true });
  await closed;

  await expect(keycloak.locator('button', { hasText: 'Logout' })).toBeVisible();
  await page.locator('.dialog-ux button.close-modal').click();

  const me = page.locator('[id="operations-Пользователь-getMe"]');
  await me.locator('.opblock-summary').click();
  await me.locator('button.try-out__btn').click();
  await me.locator('button.execute').click();
  const response = me.locator('.live-responses-table .response').first();
  await expect(response.locator('.response-col_status')).toHaveText('200');
  await expect(response).toContainText(env.DEMO_EMAIL);
});
