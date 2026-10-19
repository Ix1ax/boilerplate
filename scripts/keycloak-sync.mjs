// Приводит уже работающий Keycloak к шаблону realm (make keycloak-sync).
// Keycloak импортирует realm только при первом запуске (--import-realm пропускает существующий),
// поэтому новые пользователи, роли, брендинг «Столото ID» и настройки клиентов из
// .local/keycloak/hackathon-realm.json иначе не попадают в старый том. Скрипт идемпотентен:
// создаёт недостающее, обновляет настройки и выставляет пароли учётных записей из .env.
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const env = Object.fromEntries(
  (await readFile(path.join(root, '.env'), 'utf8'))
    .split('\n')
    .filter((line) => /^[A-Z][A-Z0-9_]*=/.test(line))
    .map((line) => {
      const i = line.indexOf('=');
      return [line.slice(0, i), line.slice(i + 1)];
    }),
);
const realm = JSON.parse(
  await readFile(
    path.join(root, '.local/keycloak/hackathon-realm.json'),
    'utf8',
  ),
);
const issuer = new URL(env.OIDC_ISSUER);
const base = process.env.KEYCLOAK_URL ?? issuer.origin;
const realmName = realm.realm;
// Пароли в шаблоне — ссылки ${ИМЯ}: Keycloak подставляет их из окружения при импорте.
const secret = (value) =>
  value.replace(/^\$\{([A-Z0-9_]+)\}$/, (_, name) => {
    if (!env[name]) throw new Error(`В .env нет ${name}: выполните make setup`);
    return env[name];
  });

const tokenResponse = await fetch(
  `${base}/realms/master/protocol/openid-connect/token`,
  {
    method: 'POST',
    body: new URLSearchParams({
      grant_type: 'password',
      client_id: 'admin-cli',
      username: env.KEYCLOAK_ADMIN,
      password: env.KEYCLOAK_ADMIN_PASSWORD,
    }),
    signal: AbortSignal.timeout(15_000),
  },
);
if (!tokenResponse.ok)
  throw new Error(
    `Keycloak не пустил администратора (${tokenResponse.status}). Он запущен? make infra`,
  );
const { access_token: token } = await tokenResponse.json();

const api = async (method, url, body) => {
  const response = await fetch(`${base}/admin/realms/${realmName}${url}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(body ? { 'Content-Type': 'application/json' } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(15_000),
  });
  if (!response.ok)
    throw new Error(
      `${method} ${url}: ${response.status} ${await response.text()}`,
    );
  const text = await response.text();
  return text ? JSON.parse(text) : null;
};

// Настройки realm, которые меняются вместе с продуктом: брендинг и регистрация.
const current = await api('GET', '');
await api('PUT', '', {
  ...current,
  displayName: realm.displayName,
  displayNameHtml: realm.displayNameHtml,
  registrationAllowed: realm.registrationAllowed,
  verifyEmail: realm.verifyEmail,
  passwordPolicy: realm.passwordPolicy,
});

for (const role of realm.roles?.realm ?? []) {
  const existing = await api('GET', '/roles').then((roles) =>
    roles.find((r) => r.name === role.name),
  );
  if (!existing) await api('POST', '/roles', { name: role.name });
}

for (const client of realm.clients) {
  const [existing] = await api(
    'GET',
    `/clients?clientId=${encodeURIComponent(client.clientId)}`,
  );
  if (!existing) {
    await api('POST', '/clients', client);
    continue;
  }
  await api('PUT', `/clients/${existing.id}`, {
    ...existing,
    directAccessGrantsEnabled: client.directAccessGrantsEnabled ?? false,
    redirectUris: client.redirectUris ?? existing.redirectUris,
    webOrigins: client.webOrigins ?? existing.webOrigins,
    attributes: { ...existing.attributes, ...client.attributes },
  });
}

let created = 0;
for (const user of realm.users ?? []) {
  const [existing] = await api(
    'GET',
    `/users?email=${encodeURIComponent(user.email)}&exact=true`,
  );
  let id = existing?.id;
  if (!id) {
    await api('POST', '/users', {
      username: user.username,
      email: user.email,
      enabled: true,
      emailVerified: true,
      firstName: user.firstName,
      lastName: user.lastName,
    });
    [{ id }] = await api(
      'GET',
      `/users?email=${encodeURIComponent(user.email)}&exact=true`,
    );
    created++;
  } else {
    await api('PUT', `/users/${id}`, {
      ...existing,
      enabled: true,
      firstName: user.firstName,
      lastName: user.lastName,
    });
  }
  for (const credential of user.credentials ?? [])
    await api('PUT', `/users/${id}/reset-password`, {
      type: 'password',
      value: secret(credential.value),
      temporary: false,
    });
  const assigned = new Set(
    (await api('GET', `/users/${id}/role-mappings/realm`)).map((r) => r.name),
  );
  const missing = [];
  for (const name of user.realmRoles ?? [])
    if (!assigned.has(name)) missing.push(await api('GET', `/roles/${name}`));
  if (missing.length)
    await api('POST', `/users/${id}/role-mappings/realm`, missing);
}
console.log(
  `Keycloak синхронизирован с шаблоном: пользователей ${realm.users?.length ?? 0} (новых ${created}), клиентов ${realm.clients.length}.`,
);
