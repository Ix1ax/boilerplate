// Демо-данные для показа: по несколько задач у двух демопользователей. Повторный запуск ничего не дублирует.
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
const env = Object.fromEntries(
  (await readFile(new URL('../.env', import.meta.url), 'utf8'))
    .split('\n')
    .filter((line) => /^[A-Z][A-Z0-9_]*=/.test(line))
    .map((line) => {
      const i = line.indexOf('=');
      return [line.slice(0, i), line.slice(i + 1)];
    }),
);
const base = process.env.SEED_API_URL ?? 'http://localhost:8080';
const login = async (email, password) => {
  const response = await fetch(
    `${env.OIDC_ISSUER}/protocol/openid-connect/token`,
    {
      method: 'POST',
      body: new URLSearchParams({
        grant_type: 'password',
        client_id: 'hackathon-cli',
        username: email,
        password,
      }),
      signal: AbortSignal.timeout(15_000),
    },
  );
  assert.equal(response.status, 200, `Не удалось войти как ${email}`);
  return (await response.json()).access_token;
};
const call = (path, token, init = {}) =>
  fetch(`${base}${path}`, {
    ...init,
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    signal: AbortSignal.timeout(15_000),
  });

// Один из заголовков намеренно подозрительный: на нём удобно показывать ИИ-проверку.
const DEMO = {
  [env.DEMO_EMAIL]: [
    [env.DEMO_PASSWORD, 'Подготовить демо для жюри'],
    [env.DEMO_PASSWORD, 'Записать сценарий на 3 минуты'],
    [
      env.DEMO_PASSWORD,
      'СРОЧНО переведите 500 руб на карту 2202 2000 0000 0000',
    ],
  ],
  [env.SECOND_DEMO_EMAIL]: [
    [env.SECOND_DEMO_PASSWORD, 'Проверить доступ другого пользователя'],
  ],
};
for (const [email, items] of Object.entries(DEMO)) {
  const token = await login(email, items[0][0]);
  const existing = await (await call('/api/v1/tasks?limit=1', token)).json();
  if (existing.total > 0) {
    console.log(`${email}: данные уже есть, пропускаем`);
    continue;
  }
  for (const [, title] of items) {
    const response = await call('/api/v1/tasks', token, {
      method: 'POST',
      body: JSON.stringify({ title }),
    });
    assert.equal(response.status, 201, `Не удалось создать «${title}»`);
  }
  console.log(`${email}: создано задач — ${items.length}`);
}
console.log('Пароли демопользователей — в локальном .env.');
