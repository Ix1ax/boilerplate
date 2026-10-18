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

// Данные берутся из fixtures/*.json и создаются через API: работают проверки, события и уведомления.
const fixture = async (name) =>
  JSON.parse(
    await readFile(new URL(`../fixtures/${name}`, import.meta.url), 'utf8'),
  ).items;
const users = {
  demo: [env.DEMO_EMAIL, env.DEMO_PASSWORD],
  second: [env.SECOND_DEMO_EMAIL, env.SECOND_DEMO_PASSWORD],
  admin: [env.ADMIN_DEMO_EMAIL, env.ADMIN_DEMO_PASSWORD],
};
const tokens = {};
const token = async (user) => {
  assert.ok(users[user], `Неизвестный пользователь в фикстуре: ${user}`);
  tokens[user] ??= await login(...users[user]);
  return tokens[user];
};
const tasks = await fixture('tasks.json');
const files = await fixture('files.json');
for (const user of Object.keys(users)) {
  const auth = await token(user);
  const existing = await (await call('/api/v1/tasks?limit=1', auth)).json();
  if (existing.total > 0) {
    console.log(`${user}: данные уже есть, пропускаем`);
    continue;
  }
  let created = 0;
  for (const item of tasks.filter((t) => t.user === user)) {
    const response = await call('/api/v1/tasks', auth, {
      method: 'POST',
      body: JSON.stringify({ title: item.title }),
    });
    assert.equal(response.status, 201, `Не удалось создать «${item.title}»`);
    const task = await response.json();
    if (item.completed)
      await call(`/api/v1/tasks/${task.id}`, auth, {
        method: 'PATCH',
        body: JSON.stringify({ completed: true }),
      });
    created++;
  }
  for (const item of files.filter((f) => f.user === user)) {
    const form = new FormData();
    const content = await readFile(
      new URL(`../fixtures/files/${item.file}`, import.meta.url),
    );
    form.append('file', new Blob([content], { type: 'text/plain' }), item.file);
    if (item.entity) {
      form.append('entity_type', item.entity.type);
      form.append('entity_id', item.entity.id);
    }
    const response = await fetch(`${base}/api/v1/files`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${auth}` },
      body: form,
      signal: AbortSignal.timeout(15_000),
    });
    assert.equal(response.status, 201, `Не удалось загрузить ${item.file}`);
    created++;
  }
  console.log(`${user}: создано записей — ${created}`);
}
console.log(
  'Пароли демопользователей — в локальном .env (make demo печатает их).',
);
