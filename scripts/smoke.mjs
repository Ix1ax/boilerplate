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
const base = process.env.SMOKE_API_URL ?? 'http://localhost:18080';
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
    'Проверка пройдена: готовность сервисов, вход, роли, валидация, задачи, права доступа, S3 и CSRF.',
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
