// Выводит ссылки и тестовых пользователей для проверяющих. Пароли берутся из локального .env.
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
console.log(`
Приложение      http://localhost:3000
Swagger (API)   http://localhost:3000/api/docs
Keycloak        http://localhost:8180   (admin: ${env.KEYCLOAK_ADMIN})
RabbitMQ        http://localhost:15672  (${env.RABBITMQ_USER})
Почта (тест)    http://localhost:8025

Тестовые пользователи:
  Роль            Email                         Пароль
  пользователь    ${env.DEMO_EMAIL.padEnd(29)} ${env.DEMO_PASSWORD}
  пользователь 2  ${env.SECOND_DEMO_EMAIL.padEnd(29)} ${env.SECOND_DEMO_PASSWORD}
  администратор   ${env.ADMIN_DEMO_EMAIL.padEnd(29)} ${env.ADMIN_DEMO_PASSWORD}
`);
