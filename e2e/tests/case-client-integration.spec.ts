import { expect, test } from '@playwright/test';

// Проверяем клиентские сценарии отдельно от доступности Docker и Keycloak.
// Серверные проверки прав и переходов покрывает make smoke:case.
test.beforeEach(async ({ page }) => {
  const sport = {
    id: '11111111-1111-4111-8111-111111111111',
    name: 'Бег',
    is_custom: false,
  };
  const event = {
    id: 'event-1',
    title: 'Совместная пробежка',
    sport,
    organizer: { id: 'user-1', name: 'Анна', type: 'person' },
    zone: 'Север',
    meeting_host: '',
    step_free: false,
    companion_allowed: false,
    starts_at: '2099-01-20T03:00:00Z',
    description: 'Пробежка в парке',
    location: 'Парк',
    level: 'any',
    join_mode: 'request',
    status: 'published',
    is_organizer: true,
    confirmed_count: 0,
    pending_count: 1,
    waitlist_count: 0,
    quorum_status: 'none',
    requirements: '',
    price_note: '',
    private_info: '',
    outdoor: false,
  };
  let organization: Record<string, unknown> | null = null;
  let requests: Record<string, unknown>[] = [];
  let participant = {
    id: 'participant-1',
    user: { id: 'user-2', name: 'Игорь', level: 'beginner' },
    status: 'pending',
    message: 'Хочу участвовать',
    decision_reason: '',
  };
  const stats = { events_held: 2, people_came: 8, upcoming_events: 0 };
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname.replace('/api/v1', '');
    const method = request.method();
    let body: unknown = { items: [], total: 0, limit: 24, offset: 0 };
    let status = 200;
    if (path === '/me') {
      body = { id: 'user-1', name: 'Анна', roles: ['user'] };
    } else if (path === '/sports') {
      body = { items: [sport] };
    } else if (path === '/my/organization') {
      if (method === 'POST' || method === 'PUT') {
        organization = {
          ...request.postDataJSON(),
          id: 'club-1',
          status: 'pending',
          sports: [sport],
          review_reason: '',
          events_count: 0,
        };
      }
      body = organization ?? {
        error: { code: 'not_found', message: 'Клуб не найден' },
      };
      if (!organization) status = 404;
    } else if (path === '/my/company-requests') {
      if (method === 'POST') {
        body = {
          ...request.postDataJSON(),
          id: 'request-1',
          sport,
          status: 'active',
          matches_count: 0,
          others_count: 2,
          expires_at: '2099-02-01T00:00:00Z',
        };
        requests = [body as Record<string, unknown>];
      } else body = { items: requests };
    } else if (
      path === '/my/company-requests/request-1' &&
      method === 'DELETE'
    ) {
      requests = [];
      status = 204;
    } else if (path.endsWith('/proposal')) {
      body = { proposal: null };
    } else if (path === '/events' && method === 'POST') {
      body = event;
    } else if (path === '/events/event-1') {
      body = event;
    } else if (path === '/events/event-1/participants') {
      body = { items: [participant] };
    } else if (path.endsWith('/approve')) {
      participant = { ...participant, status: 'confirmed' };
      body = participant;
    } else if (path.startsWith('/organizations/')) {
      body = {
        organization: {
          name: 'Ладья',
          description: 'Спорт вместе',
          city: 'Красноярск',
          sports: [sport],
          contacts: 'Пишите организатору',
        },
        stats,
        events: [],
      };
    } else if (path.startsWith('/profiles/')) {
      body = {
        name: 'Игорь',
        about: 'Играем вместе',
        city: 'Красноярск',
        level: 'beginner',
        stats,
        events: [],
      };
    }
    await route.fulfill({
      status,
      contentType: 'application/json',
      body: status === 204 ? '' : JSON.stringify(body),
    });
  });
});

test('запрос компании создаётся и закрывается', async ({ page }) => {
  await page.goto('/company');
  await page.getByLabel('Вид спорта', { exact: true }).fill('Бег');
  await page.getByLabel('Север', { exact: true }).check();
  await page.getByRole('button', { name: '+ Ещё окно', exact: true }).click();
  await page
    .getByRole('button', { name: 'Искать компанию', exact: true })
    .click();
  await expect(
    page.getByRole('heading', { name: 'Бег', exact: true }),
  ).toBeVisible();
  await page
    .getByRole('button', { name: 'Больше не ищу', exact: true })
    .click();
  await expect(
    page.getByRole('heading', { name: 'Бег', exact: true }),
  ).toHaveCount(0);
});

test('заявка клуба показывает результат отправки', async ({ page }) => {
  await page.goto('/club');
  await page.getByLabel('Название', { exact: true }).fill('Клуб друзей');
  await page
    .getByLabel('О клубе', { exact: true })
    .fill('Совместные спортивные встречи');
  await page
    .getByRole('button', { name: 'Отправить заявку', exact: true })
    .click();
  await expect(
    page.getByText('На рассмотрении', { exact: true }),
  ).toBeVisible();
  await expect(
    page
      .getByRole('main')
      .getByRole('button', { name: 'Создать встречу', exact: true }),
  ).toHaveCount(0);
});

test('встреча создаётся с новым направлением и временем Красноярска', async ({
  page,
}) => {
  await page.goto('/events/new');
  await page
    .getByLabel('Название встречи', { exact: true })
    .fill('Совместная пробежка');
  await page.getByLabel('Своё направление', { exact: true }).fill('Бег');
  await page
    .getByLabel('Начало (время Красноярска)', { exact: true })
    .fill('2099-01-20T10:00');
  await page.getByLabel('Место и адрес', { exact: true }).fill('Парк');
  const sent = page.waitForRequest(
    (request) =>
      request.url().endsWith('/api/v1/events') && request.method() === 'POST',
  );
  await page.getByRole('button', { name: 'Далее', exact: true }).click();
  await page.getByRole('button', { name: 'Далее', exact: true }).click();
  await page
    .getByRole('button', { name: 'Опубликовать встречу', exact: true })
    .click();
  const payload = (await sent).postDataJSON();
  expect(payload.sport_name).toBe('Бег');
  expect(new Date(payload.starts_at).toISOString()).toBe(
    '2099-01-20T03:00:00.000Z',
  );
  await expect(
    page.getByRole('heading', { name: 'Встреча опубликована', exact: true }),
  ).toBeVisible();
  await page
    .getByRole('link', { name: 'Открыть встречу', exact: true })
    .click();
  await expect(page).toHaveURL(/\/events\/event-1$/);
});

test('одобрение обновляет состав организатора', async ({ page }) => {
  await page.goto('/events/event-1/manage');
  await page
    .getByRole('button', { name: 'Принять заявку', exact: true })
    .click();
  await expect(
    page.getByText('Заявка принята.', { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'Принять заявку', exact: true }),
  ).toHaveCount(0);
});

test('публичные профили открываются на узком экране без переполнения', async ({
  page,
}) => {
  await page.setViewportSize({ width: 375, height: 812 });
  for (const [path, name] of [
    ['/organizations/club-1', 'Ладья'],
    ['/people/user-1', 'Игорь'],
  ]) {
    await page.goto(path);
    await expect(
      page.getByRole('heading', { name, exact: true }),
    ).toBeVisible();
    expect(
      await page
        .locator('body')
        .evaluate((body) => body.scrollWidth <= innerWidth),
    ).toBe(true);
  }
});
