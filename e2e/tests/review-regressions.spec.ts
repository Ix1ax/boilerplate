import { expect, test } from '@playwright/test';

const sport = {
  id: '11111111-1111-4111-8111-111111111111',
  name: 'Бег',
  is_custom: false,
};
const fixture = {
  id: 'event-1',
  title: 'Пробежка UTC',
  sport,
  organizer: { id: 'user-1', name: 'Анна', type: 'person' },
  starts_at: '2099-01-20T03:00:00Z',
  ends_at: '2099-01-20T04:00:00Z',
  description: 'Пробежка в парке',
  location: 'Парк Север',
  zone: 'Север',
  meeting_host: '',
  step_free: false,
  companion_allowed: false,
  level: 'any',
  join_mode: 'open',
  status: 'published',
  is_organizer: true,
  confirmed_count: 0,
  pending_count: 0,
  waitlist_count: 0,
  first_timers_count: 0,
  buddies_count: 0,
  quorum_status: 'none',
  requirements: '',
  price_note: '',
  private_info: '',
  has_private_info: false,
  outdoor: false,
  capacity: 10,
};

test('каталог загружает более ста встреч через offset без превышения limit', async ({
  page,
}) => {
  const queries: { limit: number; offset: number }[] = [];
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname.replace('/api/v1', '');
    let body: unknown = { items: [] };
    let status = 200;
    if (path === '/me') body = { id: 'user-1', name: 'Анна', roles: ['user'] };
    else if (path === '/sports') body = { items: [sport] };
    else if (path === '/events') {
      const limit = Number(url.searchParams.get('limit') ?? 24);
      const offset = Number(url.searchParams.get('offset') ?? 0);
      queries.push({ limit, offset });
      if (limit > 100) {
        status = 400;
        body = {
          error: {
            code: 'invalid_query',
            message: 'limit должен быть от 1 до 100',
          },
        };
      } else
        body = {
          items: Array.from(
            { length: Math.max(0, Math.min(limit, 130 - offset)) },
            (_, i) => ({
              ...fixture,
              id: `event-${offset + i + 1}`,
              title: `Встреча номер ${offset + i + 1}`,
            }),
          ),
          total: 130,
          limit,
          offset,
        };
    }
    await route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(body),
    });
  });
  await page.goto('/events?day=all');
  for (let count = 24; count < 130; count += 24) {
    await expect(
      page
        .getByRole('link')
        .filter({ has: page.getByText(/^Встреча номер \d+$/) }),
    ).toHaveCount(Math.min(count, 130));
    await page
      .getByRole('button', { name: 'Показать ещё', exact: true })
      .click();
  }
  await expect(
    page
      .getByRole('link')
      .filter({ has: page.getByText(/^Встреча номер \d+$/) }),
  ).toHaveCount(130);
  await expect(
    page.getByRole('button', { name: 'Показать ещё', exact: true }),
  ).toHaveCount(0);
  expect(queries.every((q) => q.limit <= 100)).toBe(true);
  expect(queries.map((q) => q.offset)).toEqual([0, 24, 48, 72, 96, 120]);
});

test('редактор переводит UTC в Красноярск и сохраняет только изменённые поля', async ({
  page,
}) => {
  let current = { ...fixture };
  let patch: Record<string, unknown> | undefined;
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname.replace('/api/v1', '');
    let body: unknown = { items: [] };
    if (path === '/me') body = { id: 'user-1', name: 'Анна', roles: ['user'] };
    else if (path === '/sports') body = { items: [sport] };
    else if (path === '/events/event-1') {
      if (route.request().method() === 'PATCH') {
        patch = route.request().postDataJSON();
        current = { ...current, ...patch };
      }
      body = current;
    }
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify(body),
    });
  });
  await page.goto('/events/event-1/manage');
  await page
    .getByRole('button', { name: 'Изменить встречу', exact: true })
    .click();
  await expect(
    page.getByLabel('Начало (время Красноярска)', { exact: true }),
  ).toHaveValue('2099-01-20T10:00');
  await expect(
    page.getByLabel('Окончание (необязательно)', { exact: true }),
  ).toHaveValue('2099-01-20T11:00');
  await page
    .getByLabel('Название встречи', { exact: true })
    .fill('Пробежка после правки');
  await page
    .getByLabel('Количество мест (пусто — без лимита)', { exact: true })
    .fill('');
  await page
    .getByRole('button', { name: 'Сохранить изменения', exact: true })
    .click();
  await expect(
    page.getByText('Изменения сохранены.', { exact: true }),
  ).toBeVisible();
  expect(patch).toEqual({ title: 'Пробежка после правки', capacity: null });
});
