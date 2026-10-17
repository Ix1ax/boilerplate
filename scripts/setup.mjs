import { randomBytes } from 'node:crypto';
import { existsSync } from 'node:fs';
import { chmod, copyFile, mkdir, readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
let text = await readFile(
  path.join(
    root,
    existsSync(path.join(root, '.env')) ? '.env' : '.env.example',
  ),
  'utf8',
);
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
  'POSTGRES_PASSWORD',
  'REDIS_PASSWORD',
  'S3_SECRET_KEY',
  'KEYCLOAK_ADMIN_PASSWORD',
  'DEMO_PASSWORD',
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
if (text.includes('REPLACE_WITH_POSTGRES_PASSWORD'))
  text = text.replace(
    'REPLACE_WITH_POSTGRES_PASSWORD',
    values.POSTGRES_PASSWORD,
  );
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
  ['DEMO_EMAIL', 'DEMO_PASSWORD', 'Демо'],
  ['SECOND_DEMO_EMAIL', 'SECOND_DEMO_PASSWORD', 'Второй'],
].map(([email, password, name]) => ({
  username: values[email],
  email: values[email],
  enabled: true,
  emailVerified: true,
  firstName: name,
  lastName: 'Пользователь',
  realmRoles: ['user'],
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
if (!existsSync(path.join(root, 'apps/web/.env.local'))) {
  const web = await readFile(path.join(root, 'apps/web/.env.example'), 'utf8');
  await writeFile(
    path.join(root, 'apps/web/.env.local'),
    web
      .replace(/^APP_ORIGIN=.*$/m, `APP_ORIGIN=${origin}`)
      .replace(/^OIDC_ISSUER=.*$/m, `OIDC_ISSUER=${values.OIDC_ISSUER}`)
      .replace(
        /^OIDC_INTERNAL_URL=.*$/m,
        `OIDC_INTERNAL_URL=${values.OIDC_ISSUER}`,
      ),
    { mode: 0o600 },
  );
}
console.log(
  'Локальные настройки готовы. Пароли находятся в .env; существующие пароли и настройки фронтенда сохранены.',
);
