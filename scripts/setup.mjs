import { randomBytes } from 'node:crypto';
import { existsSync } from 'node:fs';
import { chmod, mkdir, readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
let production = process.argv.includes('--production');
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
// Повторный make setup на рабочем окружении не должен вернуть тестовых пользователей.
production ||= values.APP_ENV === 'production';
const setValue = (name, value) => {
  values[name] = value;
  text = new RegExp(`^${name}=`, 'm').test(text)
    ? text.replace(new RegExp(`^${name}=.*$`, 'm'), `${name}=${value}`)
    : `${text}\n${name}=${value}\n`;
};
if (production) {
  for (const name of ['APP_DOMAIN', 'AUTH_DOMAIN']) {
    const domain = values[name] || '';
    if (
      domain.length > 253 ||
      !domain.includes('.') ||
      !domain
        .split('.')
        .every((label) => /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/i.test(label))
    )
      throw new Error(`Задайте ${name}: доменное имя без протокола и пути`);
  }
  if (values.APP_DOMAIN === values.AUTH_DOMAIN)
    throw new Error('APP_DOMAIN и AUTH_DOMAIN должны различаться');
  setValue('APP_ENV', 'production');
  setValue('APP_ORIGIN', `https://${values.APP_DOMAIN}`);
  setValue('OIDC_ISSUER', `https://${values.AUTH_DOMAIN}/realms/hackathon`);
  setValue('SESSION_COOKIE_SECURE', 'true');
}
for (const name of [
  'TASKS_DB_PASSWORD',
  'FILES_DB_PASSWORD',
  'NOTIFICATIONS_DB_PASSWORD',
  'AUDIT_DB_PASSWORD',
  'EVENTS_DB_PASSWORD',
  'MODERATION_DB_PASSWORD',
  'IDENTITY_DB_PASSWORD',
  'KEYCLOAK_DB_PASSWORD',
  'RABBITMQ_PASSWORD',
  'REDIS_PASSWORD',
  'INTERNAL_TOKEN',
  'S3_SECRET_KEY',
  'KEYCLOAK_ADMIN_PASSWORD',
  // Суперадминистратор админки нужен и на рабочем окружении: без него в админку войти некому.
  'SUPERADMIN_PASSWORD',
  ...(!production
    ? [
        'DEMO_PASSWORD',
        'SECOND_DEMO_PASSWORD',
        'ORG_DEMO_PASSWORD',
        'ORG_PENDING_DEMO_PASSWORD',
        'ORG_REJECTED_DEMO_PASSWORD',
      ]
    : []),
  'MAILPIT_PASSWORD',
  ...(production ? ['METRICS_TOKEN'] : []),
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
// Keycloak работает другим UID; каталог импорта содержит только шаблон, секреты — в окружении.
await chmod(path.join(root, '.local/keycloak'), 0o755);
const realm = JSON.parse(
  await readFile(path.join(root, 'infra/keycloak/realm.template.json'), 'utf8'),
);
if (production) {
  realm.sslRequired = 'external';
  realm.clients = realm.clients.filter(
    (client) => client.clientId !== 'hackathon-cli',
  );
}
const origin = new URL(values.APP_ORIGIN).origin;
for (const client of realm.clients) {
  if (client.clientId === 'hackathon-web') {
    // «Столото ID» — только перенаправление на страницу входа (PKCE). Пароли SSO-пользователей
    // gateway не видит: вход по паролю в приложении идёт через свои учётные записи (identity).
    client.directAccessGrantsEnabled = false;
    client.redirectUris = [`${origin}/auth/callback`];
    client.webOrigins = [origin];
    client.attributes['post.logout.redirect.uris'] = `${origin}/`;
  }
  // Отдельный клиент для входа в Swagger UI: не влияет на вход приложения.
  // Локально Swagger открывают и через фронтенд (:3000), и прямо на gateway (:8080).
  if (client.clientId === 'hackathon-swagger') {
    const origins = [origin];
    if (['localhost', '127.0.0.1'].includes(new URL(origin).hostname))
      origins.push('http://localhost:8080');
    client.redirectUris = origins.map(
      (o) => `${o}/api/docs/oauth2-redirect.html`,
    );
    client.webOrigins = origins;
  }
}
// Рабочий realm начинается без тестовых пользователей. Администраторов в Keycloak нет совсем:
// у админки свой вход (SUPERADMIN_EMAIL и SUPERADMIN_PASSWORD, сервис identity).
realm.users = production
  ? []
  : [
      ['DEMO_EMAIL', 'DEMO_PASSWORD', 'Анна', 'Смирнова', ['user']],
      [
        'SECOND_DEMO_EMAIL',
        'SECOND_DEMO_PASSWORD',
        'Игорь',
        'Волков',
        ['user'],
      ],
      ['ORG_DEMO_EMAIL', 'ORG_DEMO_PASSWORD', 'Олег', 'Ладьин', ['user']],
      [
        'ORG_PENDING_DEMO_EMAIL',
        'ORG_PENDING_DEMO_PASSWORD',
        'Мария',
        'Соснова',
        ['user'],
      ],
      [
        'ORG_REJECTED_DEMO_EMAIL',
        'ORG_REJECTED_DEMO_PASSWORD',
        'Пётр',
        'Пульсов',
        ['user'],
      ],
    ].map(([email, password, firstName, lastName, realmRoles]) => ({
      username: values[email],
      email: values[email],
      enabled: true,
      emailVerified: true,
      firstName,
      lastName,
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
const webPath = path.join(root, 'frontend/client/.env.local');
const webTemplate = await readFile(
  path.join(root, 'frontend/client/.env.example'),
  'utf8',
);
const web = existsSync(webPath)
  ? addMissing(await readFile(webPath, 'utf8'), webTemplate)
  : webTemplate;
await writeFile(webPath, web, { mode: 0o600 });
await chmod(webPath, 0o600);
console.log(
  `${production ? 'Серверные' : 'Локальные'} настройки готовы. Пароли находятся в .env; существующие пароли и настройки фронтенда сохранены.`,
);
