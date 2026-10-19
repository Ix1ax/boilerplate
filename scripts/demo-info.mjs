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

Учётные записи «Столото ID» (кнопка «Войти через Столото ID»). Свою учётную запись
любой может завести регистрацией по почте и паролю на экране входа.
  Роль                          Email                         Пароль
  пользователь (Анна)           ${env.DEMO_EMAIL.padEnd(29)} ${env.DEMO_PASSWORD}
  пользователь (Игорь)          ${env.SECOND_DEMO_EMAIL.padEnd(29)} ${env.SECOND_DEMO_PASSWORD}
  организация допущена (Олег)   ${env.ORG_DEMO_EMAIL.padEnd(29)} ${env.ORG_DEMO_PASSWORD}
  организация ждёт (Мария)      ${env.ORG_PENDING_DEMO_EMAIL.padEnd(29)} ${env.ORG_PENDING_DEMO_PASSWORD}
  организация отклонена (Пётр)  ${env.ORG_REJECTED_DEMO_EMAIL.padEnd(29)} ${env.ORG_REJECTED_DEMO_PASSWORD}

Админка         http://localhost:3000/admin/   — отдельный вход, не «Столото ID»
  суперадминистратор            ${env.SUPERADMIN_EMAIL.padEnd(29)} ${env.SUPERADMIN_PASSWORD}
  Он заводит остальных администраторов в разделе «Администраторы».
`);
