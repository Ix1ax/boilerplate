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
const base = process.env.SMOKE_API_URL ?? 'http://localhost:8080';
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
  assert.equal(
    response.status,
    200,
    'Вход локального клиента для проверки API',
  );
  return (await response.json()).access_token;
};
const webHealth = await fetch(`${env.APP_ORIGIN}/healthz`);
assert.equal(webHealth.status, 200);
const deepLink = await fetch(`${env.APP_ORIGIN}/dashboard/files`);
assert.equal(deepLink.status, 200);
assert.match(await deepLink.text(), /<div id="root"><\/div>/);
const startedLogin = await fetch(`${env.APP_ORIGIN}/auth/login`, {
  redirect: 'manual',
});
assert.equal(startedLogin.status, 302);
const authorization = new URL(startedLogin.headers.get('location'));
assert.equal(authorization.searchParams.get('code_challenge_method'), 'S256');
assert.ok(authorization.searchParams.get('state'));
assert.match(startedLogin.headers.get('set-cookie'), /HttpOnly/i);
const invalidCallback = await fetch(
  `${env.APP_ORIGIN}/auth/callback?code=invalid&state=invalid`,
);
assert.equal(invalidCallback.status, 400);
const invalidLogout = await fetch(`${env.APP_ORIGIN}/auth/logout`, {
  method: 'POST',
  headers: { Origin: 'https://untrusted.example' },
});
assert.equal(invalidLogout.status, 403);
const owner = await login(env.DEMO_EMAIL, env.DEMO_PASSWORD);
const other = await login(env.SECOND_DEMO_EMAIL, env.SECOND_DEMO_PASSWORD);
const request = (path, token, init = {}) =>
  fetch(`${base}${path}`, {
    ...init,
    headers: {
      ...(init.body instanceof FormData
        ? {}
        : { 'Content-Type': 'application/json' }),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init.headers,
    },
    signal: AbortSignal.timeout(20_000),
  });
assert.equal((await request('/readyz')).status, 200);
assert.equal((await request('/api/v1/me')).status, 401);
assert.equal((await request('/api/v1/me', owner)).status, 200);
assert.equal((await request('/api/v1/files?limit=0', owner)).status, 400);
// LLM-шлюз: статус доступен всегда, генерация без модели даёт 503, но не 5xx-падение.
assert.equal((await request('/api/v1/llm/status', owner)).status, 200);
assert.equal((await fetch(`${base}/api/v1/llm/status`)).status, 401);
// Асинхронная проверка через RabbitMQ: outbox → брокер → потребитель → результат в списке задач.
{
  const created = await request('/api/v1/tasks', owner, {
    method: 'POST',
    body: JSON.stringify({
      title: 'Переведите 500 руб на карту 2202 2000 0000 0000',
    }),
  });
  assert.equal(created.status, 201);
  const risky = await created.json();
  let moderation = null;
  for (let i = 0; i < 30 && !moderation; i++) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    const page = await (await request('/api/v1/tasks?limit=100', owner)).json();
    moderation = page.items.find((item) => item.id === risky.id)?.moderation;
  }
  await request(`/api/v1/tasks/${risky.id}`, owner, { method: 'DELETE' });
  assert.ok(moderation, 'Проверка через RabbitMQ не выполнена за 30 секунд');
  assert.equal(moderation.verdict, 'needs_review');
}
// Межсервисное событие: tasks → notification.requested → notifications-service → колокольчик.
{
  let found = false;
  for (let i = 0; i < 20 && !found; i++) {
    const page = await (await request('/api/v1/notifications', owner)).json();
    found = page.items.some((item) => item.title.startsWith('ИИ-проверка'));
    if (!found) await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  assert.ok(found, 'Уведомление через RabbitMQ не пришло за 20 секунд');
  assert.equal(
    (await request('/api/v1/notifications/read-all', owner, { method: 'POST' }))
      .status,
    204,
  );
  assert.equal(
    (await (await request('/api/v1/notifications', owner)).json()).unread,
    0,
  );
}
// Live-уведомление по SSE и письмо в Mailpit для новой задачи.
{
  const before = await (
    await fetch('http://localhost:8025/api/v1/messages')
  ).json();
  const stream = await fetch(`${base}/api/v1/notifications/stream`, {
    headers: { Authorization: `Bearer ${owner}` },
    signal: AbortSignal.timeout(30_000),
  });
  assert.equal(stream.status, 200);
  const reader = stream.body.pipeThrough(new TextDecoderStream()).getReader();
  await reader.read(); // «: connected» — подписка активна до создания задачи
  const created = await request('/api/v1/tasks', owner, {
    method: 'POST',
    body: JSON.stringify({ title: 'Звони +7 999 123-45-67 срочно' }),
  });
  const live = await created.json();
  let text = '';
  while (!text.includes('event: notification')) {
    const { value, done } = await reader.read();
    if (done) break;
    text += value;
  }
  await reader.cancel();
  await request(`/api/v1/tasks/${live.id}`, owner, { method: 'DELETE' });
  assert.ok(
    text.includes('event: notification'),
    'Нет live-уведомления по SSE',
  );
  const after = await (
    await fetch('http://localhost:8025/api/v1/messages')
  ).json();
  assert.ok(
    after.messages_count > before.messages_count,
    'Письмо не дошло до Mailpit',
  );
}
// Idempotency-Key: повтор того же создания не создаёт дубль и помечается заголовком.
{
  const key = `smoke-${Date.now()}`;
  const send = () =>
    request('/api/v1/tasks', owner, {
      method: 'POST',
      headers: { 'Idempotency-Key': key },
      body: JSON.stringify({ title: 'Проверка идемпотентности' }),
    });
  const first = await send();
  const second = await send();
  const [a, b] = [await first.json(), await second.json()];
  assert.equal(second.headers.get('idempotent-replayed'), 'true');
  assert.equal(a.id, b.id, 'Повтор с тем же ключом создал вторую задачу');
  await request(`/api/v1/tasks/${a.id}`, owner, { method: 'DELETE' });
}
// Админская выгрузка: обычному пользователю запрещена, администратору отдаёт CSV для Excel.
{
  assert.equal(
    (await request('/api/v1/admin/tasks/export.csv', owner)).status,
    403,
  );
  const admin = await login(env.ADMIN_DEMO_EMAIL, env.ADMIN_DEMO_PASSWORD);
  const csv = await request('/api/v1/admin/tasks/export.csv', admin);
  assert.equal(csv.status, 200);
  assert.match(csv.headers.get('content-type') ?? '', /text\/csv/);
  // text() отбрасывает BOM, поэтому проверяем байты.
  const bytes = Buffer.from(await csv.arrayBuffer());
  assert.ok(
    bytes.subarray(0, 3).equals(Buffer.from([0xef, 0xbb, 0xbf])) &&
      bytes.subarray(3, 6).toString() === 'ID;',
    'CSV должен начинаться с BOM и заголовка',
  );
}
assert.equal((await request('/api/v1/admin/ping', owner)).status, 403);
assert.equal(
  (
    await request('/api/v1/tasks', owner, {
      method: 'POST',
      body: JSON.stringify({ title: ' ' }),
    })
  ).status,
  400,
);
let task;
let file;
try {
  const created = await request('/api/v1/tasks', owner, {
    method: 'POST',
    body: JSON.stringify({ title: 'Проверка API' }),
  });
  assert.equal(created.status, 201);
  task = await created.json();
  assert.equal(
    (
      await request(`/api/v1/tasks/${task.id}`, other, {
        method: 'PATCH',
        body: JSON.stringify({ completed: true }),
      })
    ).status,
    404,
  );
  assert.equal(
    (await request(`/api/v1/tasks/${task.id}`, other, { method: 'DELETE' }))
      .status,
    404,
  );
  const patched = await request(`/api/v1/tasks/${task.id}`, owner, {
    method: 'PATCH',
    body: JSON.stringify({ completed: true }),
  });
  assert.equal(patched.status, 200);
  assert.equal((await patched.json()).completed, true);
  const listing = await request('/api/v1/tasks?limit=100', owner);
  assert.equal(listing.status, 200);
  assert.ok((await listing.json()).items.some((item) => item.id === task.id));
  const form = new FormData();
  form.append(
    'file',
    new Blob(['Проверка файлового хранилища'], { type: 'text/plain' }),
    'smoke.txt',
  );
  const uploaded = await request('/api/v1/files', owner, {
    method: 'POST',
    body: form,
  });
  assert.equal(uploaded.status, 201);
  file = await uploaded.json();
  assert.equal((await request(`/api/v1/files/${file.id}`, other)).status, 404);
  const ownFiles = await request('/api/v1/files?limit=100', owner);
  assert.equal(ownFiles.status, 200);
  assert.ok((await ownFiles.json()).items.some((item) => item.id === file.id));
  const otherFiles = await request('/api/v1/files?limit=100', other);
  assert.equal(otherFiles.status, 200);
  assert.ok(
    !(await otherFiles.json()).items.some((item) => item.id === file.id),
  );
  assert.equal(
    (await request(`/api/v1/files/${file.id}`, other, { method: 'DELETE' }))
      .status,
    404,
  );
  const downloaded = await request(`/api/v1/files/${file.id}`, owner);
  assert.equal(downloaded.status, 200);
  assert.equal(await downloaded.text(), 'Проверка файлового хранилища');
  const csrf = await fetch(`${env.APP_ORIGIN}/api/v1/tasks`, {
    method: 'POST',
    headers: {
      Origin: 'https://untrusted.example',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ title: 'blocked' }),
  });
  assert.equal(csrf.status, 403);
  console.log(
    'Проверка пройдена: готовность, SPA, OIDC/PKCE, роли, валидация, задачи, список и приватность файлов, S3, CSRF, события RabbitMQ, уведомления (SSE и почта), идемпотентность и выгрузка CSV.',
  );
} finally {
  if (file)
    assert.equal(
      (await request(`/api/v1/files/${file.id}`, owner, { method: 'DELETE' }))
        .status,
      204,
    );
  if (task)
    assert.equal(
      (await request(`/api/v1/tasks/${task.id}`, owner, { method: 'DELETE' }))
        .status,
      204,
    );
}
