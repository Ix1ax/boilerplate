import { randomBytes } from 'node:crypto';
import { existsSync } from 'node:fs';
import { chmod, mkdir, readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const addMissing = (existing, template) => {
  let merged = existing;
  for (const line of template.split('\n')) {
    const key = line.match(/^([A-Z][A-Z0-9_]*)=/)?.[1];
    if (key && !new RegExp(`^${key}=`, 'm').test(merged)) {
      merged += `${merged.endsWith('\n') ? '' : '\n'}${line}\n`;
    }
  }
  return merged;
};
const template = await readFile(path.join(root, '.env.example'), 'utf8');
let text = existsSync(path.join(root, '.env'))
  ? addMissing(await readFile(path.join(root, '.env'), 'utf8'), template)
  : template;
const values = Object.fromEntries(
  text
    .split('\n')
    .filter((line) => /^[A-Z][A-Z0-9_]*=/.test(line))
    .map((line) => {
      const i = line.indexOf('=');
      return [line.slice(0, i), line.slice(i + 1)];
    }),
);
for (const name of [
  'TASKS_DB_PASSWORD',
  'FILES_DB_PASSWORD',
  'NOTIFICATIONS_DB_PASSWORD',
  'AUDIT_DB_PASSWORD',
  'KEYCLOAK_DB_PASSWORD',
  'RABBITMQ_PASSWORD',
  'REDIS_PASSWORD',
  'INTERNAL_TOKEN',
  'S3_SECRET_KEY',
  'KEYCLOAK_ADMIN_PASSWORD',
  'DEMO_PASSWORD',
  'ADMIN_DEMO_PASSWORD',
  'SECOND_DEMO_PASSWORD',
]) {
  if (!values[name]) {
    values[name] = randomBytes(24).toString('hex');
    text = text.replace(
      new RegExp(`^${name}=.*$`, 'm'),
      `${name}=${values[name]}`,
    );
  }
}
// Подставляем сгенерированные пароли в строки подключения (REPLACE_WITH_<ИМЯ_ПАРОЛЯ>).
for (const [name, value] of Object.entries(values))
  text = text.replaceAll(`REPLACE_WITH_${name}`, value);
await writeFile(path.join(root, '.env'), text, { mode: 0o600 });
await chmod(path.join(root, '.env'), 0o600);
await mkdir(path.join(root, '.local/keycloak'), { recursive: true });
const realm = JSON.parse(
  await readFile(path.join(root, 'infra/keycloak/realm.template.json'), 'utf8'),
);
const origin = new URL(values.APP_ORIGIN).origin;
for (const client of realm.clients) {
  if (client.clientId === 'hackathon-web') {
    client.redirectUris = [`${origin}/auth/callback`];
    client.webOrigins = [origin];
    client.attributes['post.logout.redirect.uris'] = `${origin}/`;
  }
}
realm.users = [
  ['DEMO_EMAIL', 'DEMO_PASSWORD', 'Демо', ['user']],
  ['SECOND_DEMO_EMAIL', 'SECOND_DEMO_PASSWORD', 'Второй', ['user']],
  ['ADMIN_DEMO_EMAIL', 'ADMIN_DEMO_PASSWORD', 'Админ', ['user', 'admin']],
].map(([email, password, name, realmRoles]) => ({
  username: values[email],
  email: values[email],
  enabled: true,
  emailVerified: true,
  firstName: name,
  lastName: 'Пользователь',
  realmRoles,
  credentials: [
    { type: 'password', value: '${' + password + '}', temporary: false },
  ],
}));
await writeFile(
  path.join(root, '.local/keycloak/hackathon-realm.json'),
  JSON.stringify(realm, null, 2) + '\n',
  { mode: 0o644 },
);
await chmod(path.join(root, '.local/keycloak/hackathon-realm.json'), 0o644);
const webPath = path.join(root, 'client/.env.local');
const webTemplate = await readFile(
  path.join(root, 'client/.env.example'),
  'utf8',
);
const web = existsSync(webPath)
  ? addMissing(await readFile(webPath, 'utf8'), webTemplate)
  : webTemplate;
await writeFile(webPath, web, { mode: 0o600 });
await chmod(webPath, 0o600);
console.log(
  'Локальные настройки готовы. Пароли находятся в .env; существующие пароли и настройки фронтенда сохранены.',
);
