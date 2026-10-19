import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import {
  cpSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readlinkSync,
  readdirSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs';
import path from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';

const repository = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
);
const revision = 'a'.repeat(40);

// Подменяем только внешние операции: распаковка, файлы и управление версиями остаются настоящими.
const fakeCommand = `#!${process.execPath}
const fs = require('node:fs');
const path = require('node:path');
const name = path.basename(process.argv[1]);
const args = process.argv.slice(2);
const root = process.env.TEST_DEPLOY_ROOT;
const current = path.join(root, 'current');
const marker = path.join(process.cwd(), '.revision');
fs.appendFileSync(process.env.TEST_TRACE, JSON.stringify({
  name, args, cwd: process.cwd(),
  current: fs.existsSync(current) ? fs.readlinkSync(current) : null,
  revision: fs.existsSync(marker) ? fs.readFileSync(marker, 'utf8').trim() : null,
}) + '\\n');
if (name === 'mv') {
  if (args[0] !== '-Tf') process.exit(80);
  fs.renameSync(args[1], args[2]);
}
if (name === 'docker') {
  // Задания уже exited: без --all Compose не вернёт их идентификаторы.
  if (args[0] === 'compose' && args.includes('ps') && args.includes('--all') && args.includes('--quiet')) {
    process.stdout.write('exited-' + args.at(-1) + '\\n');
  }
  // docker wait успешно исполняется и печатает код контейнера, включая ненулевой.
  if (args[0] === 'wait') {
    process.stdout.write(args[1] === 'exited-' + process.env.TEST_FAIL_MIGRATION ? '42\\n' : '0\\n');
  }
  if (args.includes('exec') && args.some((arg) => arg.includes('pg_dump'))) process.stdout.write('test-database-backup');
}
if (name === 'curl' && process.env.TEST_FAIL_PUBLIC && args.includes('https://app.example.invalid/readyz')) process.exit(22);
`;

function fixture(t, extraEnv = {}) {
  // Входной путь специально без точек: deploy.sh ограничивает допустимые имена каталога.
  const directory = mkdtempSync('/tmp/deployverify');
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const root = path.join(directory, 'server');
  const source = path.join(directory, 'source');
  const bin = path.join(directory, 'bin');
  const archive = path.join(directory, 'release.tar.gz');
  const trace = path.join(directory, 'trace.jsonl');
  for (const folder of [root, source, bin]) mkdirSync(folder);
  writeFileSync(
    path.join(root, '.env'),
    'COMPOSE_PROJECT_NAME=regression\nAPP_DOMAIN=app.example.invalid\nAUTH_DOMAIN=auth.example.invalid\n',
  );
  writeFileSync(path.join(root, '.project-name'), 'regression\n');
  const oldRelease = path.join(root, 'previous');
  mkdirSync(oldRelease);
  writeFileSync(path.join(oldRelease, '.revision'), 'previous\n');
  symlinkSync(oldRelease, path.join(root, 'current'));
  for (const file of ['compose.yaml', 'compose.prod.yaml', '.env.example'])
    writeFileSync(path.join(source, file), '# Тестовый архив\n');
  const packed = spawnSync('tar', ['-czf', archive, '-C', source, '.'], {
    encoding: 'utf8',
  });
  assert.equal(packed.status, 0, packed.stderr);
  for (const command of ['docker', 'curl', 'flock', 'mv'])
    writeFileSync(path.join(bin, command), fakeCommand, { mode: 0o755 });
  const run = (sha = revision) =>
    spawnSync(
      'bash',
      [path.join(repository, 'deploy/deploy.sh'), root, sha, archive],
      {
        encoding: 'utf8',
        env: {
          ...process.env,
          PATH: `${bin}:${process.env.PATH}`,
          TEST_DEPLOY_ROOT: root,
          TEST_TRACE: trace,
          ...extraEnv,
        },
      },
    );
  const commands = () =>
    existsSync(trace)
      ? readFileSync(trace, 'utf8')
          .trim()
          .split('\n')
          .filter(Boolean)
          .map((line) => JSON.parse(line))
      : [];
  return { root, oldRelease, run, commands };
}

function composeOperation(command, verb) {
  return (
    command.name === 'docker' &&
    command.args[0] === 'compose' &&
    command.args.includes(verb)
  );
}

test('деплой собирает все приложения и ждёт уже завершившиеся задания до переключения версии', (t) => {
  const fixtureData = fixture(t);
  const { root, oldRelease, run, commands } = fixtureData;
  const result = run();
  assert.equal(result.status, 0, result.stderr);
  const calls = commands();
  const builds = calls
    .filter((call) => composeOperation(call, 'build'))
    .map((call) => call.args.at(-1));
  assert.deepEqual(builds, [
    'tasks-service',
    'files-service',
    'notifications-service',
    'audit-service',
    'events-service',
    'moderation-service',
    'identity-service',
    'realtime-service',
    'llm-service',
    'gateway-service',
    'admin-service',
    'web-service',
  ]);

  const backups = readdirSync(path.join(root, 'backups'));
  assert.equal(backups.length, 1);
  const backupDirectory = path.join(root, 'backups', backups[0]);
  assert.deepEqual(readdirSync(backupDirectory).sort(), [
    'audit.dump',
    'events.dump',
    'files.dump',
    'identity.dump',
    'keycloak.dump',
    'moderation.dump',
    'notifications.dump',
    'tasks.dump',
  ]);
  for (const file of readdirSync(backupDirectory))
    assert.equal(
      readFileSync(path.join(backupDirectory, file), 'utf8'),
      'test-database-backup',
    );

  const rolloutIndex = calls.findIndex(
    (call) =>
      composeOperation(call, 'up') &&
      call.args.includes('--wait-timeout') &&
      call.args.includes('900'),
  );
  assert.ok(rolloutIndex > 0);
  const lastBackup = calls.findLastIndex(
    (call) =>
      composeOperation(call, 'exec') &&
      call.args.some((arg) => arg.includes('pg_dump')),
  );
  let lastWait = lastBackup;
  for (const service of [
    'tasks-migrate',
    'files-migrate',
    'notifications-migrate',
    'audit-migrate',
    'events-migrate',
    'moderation-migrate',
    'identity-migrate',
    'llm-ollama-pull',
  ]) {
    const recreateIndex = calls.findIndex(
      (call) => composeOperation(call, 'up') && call.args.at(-1) === service,
    );
    const containerIndex = calls.findIndex(
      (call) => composeOperation(call, 'ps') && call.args.at(-1) === service,
    );
    const waitIndex = calls.findIndex(
      (call) =>
        call.name === 'docker' &&
        call.args[0] === 'wait' &&
        call.args[1] === `exited-${service}`,
    );
    assert.ok(
      recreateIndex > lastWait &&
        containerIndex > recreateIndex &&
        waitIndex > containerIndex &&
        waitIndex < rolloutIndex,
      service,
    );
    for (const flag of ['--all', '--quiet'])
      assert.ok(
        calls[containerIndex].args.includes(flag),
        `${service}: ${flag}`,
      );
    for (const flag of ['--no-deps', '--no-build', '--force-recreate'])
      assert.ok(
        calls[recreateIndex].args.includes(flag),
        `${service}: ${flag}`,
      );
    lastWait = waitIndex;
  }

  const webRestart = calls.findIndex(
    (call) =>
      composeOperation(call, 'up') && call.args.at(-1) === 'web-service',
  );
  const internalProbe = calls.findIndex(
    (call) =>
      composeOperation(call, 'exec') &&
      call.args.includes('http://gateway-service:8080/readyz'),
  );
  const publicProbes = calls.filter((call) => call.name === 'curl');
  assert.ok(webRestart > rolloutIndex && internalProbe > webRestart);
  assert.deepEqual(
    publicProbes.map((call) =>
      call.args.find((arg) => arg.startsWith('https://')),
    ),
    [
      'https://app.example.invalid/healthz',
      'https://app.example.invalid/readyz',
      'https://auth.example.invalid/realms/hackathon/.well-known/openid-configuration',
    ],
  );
  assert.ok(calls.indexOf(publicProbes[0]) > internalProbe);
  for (const call of publicProbes) {
    assert.equal(call.current, oldRelease);
    assert.equal(call.revision, null);
  }
  const current = readlinkSync(path.join(root, 'current'));
  assert.notEqual(current, oldRelease);
  assert.equal(
    readFileSync(path.join(current, '.revision'), 'utf8').trim(),
    revision,
  );
  assert.equal(calls.at(-1).name, 'mv');
  assert.equal(calls.at(-1).revision, revision);
  assert.equal(calls.at(-1).current, oldRelease);
});

test('ненулевой код контейнера из stdout docker wait останавливает деплой', (t) => {
  const { root, oldRelease, run, commands } = fixture(t, {
    TEST_FAIL_MIGRATION: 'files-migrate',
  });
  const result = run();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /files-migrate завершился с кодом 42/);
  assert.equal(readlinkSync(path.join(root, 'current')), oldRelease);
  assert.ok(
    commands().some(
      (call) =>
        call.name === 'docker' &&
        call.args[0] === 'wait' &&
        call.args[1] === 'exited-files-migrate',
    ),
  );
  assert.ok(
    !commands().some(
      (call) => composeOperation(call, 'up') && call.args.includes('900'),
    ),
  );
  assert.ok(
    !commands().some((call) => call.name === 'curl' || call.name === 'mv'),
  );
  for (const folder of readdirSync(path.join(root, 'releases')))
    assert.ok(!existsSync(path.join(root, 'releases', folder, '.revision')));
});

test('неуспешная публичная проверка не помечает версию как успешную', (t) => {
  const { root, oldRelease, run, commands } = fixture(t, {
    TEST_FAIL_PUBLIC: '1',
  });
  const result = run();
  assert.notEqual(result.status, 0);
  assert.equal(readlinkSync(path.join(root, 'current')), oldRelease);
  assert.equal(commands().filter((call) => call.name === 'curl').length, 2);
  assert.ok(!commands().some((call) => call.name === 'mv'));
  for (const folder of readdirSync(path.join(root, 'releases')))
    assert.ok(!existsSync(path.join(root, 'releases', folder, '.revision')));
});

test('смена имени Compose-проекта отклоняется до изменения контейнеров', (t) => {
  const { root, oldRelease, run, commands } = fixture(t);
  writeFileSync(path.join(root, '.project-name'), 'anotherproject\n');
  const result = run();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /COMPOSE_PROJECT_NAME изменён/);
  assert.equal(readlinkSync(path.join(root, 'current')), oldRelease);
  assert.ok(
    !commands().some((call) =>
      ['build', 'up', 'pull', 'exec'].some((verb) =>
        composeOperation(call, verb),
      ),
    ),
  );
});

test('неверный SHA отклоняется до вызова Docker', (t) => {
  const { run, commands } = fixture(t);
  const result = run('main');
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /некорректный SHA/);
  assert.deepEqual(commands(), []);
});

function setupFixture(t) {
  const root = mkdtempSync('/tmp/setupverify');
  t.after(() => rmSync(root, { recursive: true, force: true }));
  for (const file of [
    'scripts/setup.mjs',
    '.env.example',
    'frontend/client/.env.example',
    'infra/keycloak/realm.template.json',
  ]) {
    mkdirSync(path.dirname(path.join(root, file)), { recursive: true });
    cpSync(path.join(repository, file), path.join(root, file));
  }
  const template = readFileSync(path.join(root, '.env.example'), 'utf8');
  const environment = template
    .replace(/^APP_DOMAIN=.*$/m, 'APP_DOMAIN=app.example.invalid')
    .replace(/^AUTH_DOMAIN=.*$/m, 'AUTH_DOMAIN=auth.example.invalid');
  writeFileSync(path.join(root, '.env'), environment);
  const run = () =>
    spawnSync(
      process.execPath,
      [path.join(root, 'scripts/setup.mjs'), '--production'],
      { encoding: 'utf8' },
    );
  return { root, environment, run };
}

test('production setup создаёт HTTPS-настройки и сохраняет секреты при повторном запуске', (t) => {
  const { root, run } = setupFixture(t);
  const result = run();
  assert.equal(result.status, 0, result.stderr);
  const environment = readFileSync(path.join(root, '.env'), 'utf8');
  for (const line of [
    'APP_ENV=production',
    'APP_ORIGIN=https://app.example.invalid',
    'OIDC_ISSUER=https://auth.example.invalid/realms/hackathon',
    'SESSION_COOKIE_SECURE=true',
  ])
    assert.ok(environment.split('\n').includes(line), line);
  assert.match(environment, /^METRICS_TOKEN=[a-f0-9]{48}$/m);
  const realm = JSON.parse(
    readFileSync(
      path.join(root, '.local/keycloak/hackathon-realm.json'),
      'utf8',
    ),
  );
  assert.equal(realm.sslRequired, 'external');
  assert.ok(
    !realm.clients.some((client) => client.clientId === 'hackathon-cli'),
  );
  const web = realm.clients.find(
    (client) => client.clientId === 'hackathon-web',
  );
  assert.deepEqual(web.redirectUris, [
    'https://app.example.invalid/auth/callback',
  ]);
  assert.deepEqual(web.webOrigins, ['https://app.example.invalid']);
  assert.equal(
    web.attributes['post.logout.redirect.uris'],
    'https://app.example.invalid/',
  );
  const swagger = realm.clients.find(
    (client) => client.clientId === 'hackathon-swagger',
  );
  assert.deepEqual(swagger.redirectUris, [
    'https://app.example.invalid/api/docs/oauth2-redirect.html',
  ]);
  assert.deepEqual(realm.users, []);
  assert.doesNotMatch(environment, /^DEMO_PASSWORD=.+$/m);
  assert.doesNotMatch(environment, /^ADMIN_DEMO_PASSWORD=.+$/m);
  assert.equal(run().status, 0);
  assert.equal(readFileSync(path.join(root, '.env'), 'utf8'), environment);
  const noFlag = spawnSync(process.execPath, [path.join(root, 'scripts/setup.mjs')], { encoding: 'utf8' });
  assert.equal(noFlag.status, 0, noFlag.stderr);
  const repeatedRealm = JSON.parse(readFileSync(path.join(root, '.local/keycloak/hackathon-realm.json'), 'utf8'));
  assert.deepEqual(repeatedRealm.users, []);
});

test('production setup отклоняет некорректные домены, сохраняя исходный .env', (t) => {
  const { root, environment, run } = setupFixture(t);
  for (const domain of [
    'https://app.example.invalid',
    'app.example.invalid/path',
    'localhost',
    '-app.example.invalid',
    'auth.example.invalid',
  ]) {
    const invalid = environment.replace(
      'APP_DOMAIN=app.example.invalid',
      `APP_DOMAIN=${domain}`,
    );
    writeFileSync(path.join(root, '.env'), invalid);
    assert.notEqual(run().status, 0, domain);
    assert.equal(readFileSync(path.join(root, '.env'), 'utf8'), invalid);
    assert.ok(
      !existsSync(path.join(root, '.local/keycloak/hackathon-realm.json')),
    );
  }
});
