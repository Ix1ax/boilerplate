// Демо-данные для показа: мероприятия, организации, заявки и обсуждения кейса (fixtures/events.json),
// а также задачи и файлы образца шаблона. Повторный запуск ничего не дублирует.
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
  org: [env.ORG_DEMO_EMAIL, env.ORG_DEMO_PASSWORD],
  org_pending: [env.ORG_PENDING_DEMO_EMAIL, env.ORG_PENDING_DEMO_PASSWORD],
  org_rejected: [env.ORG_REJECTED_DEMO_EMAIL, env.ORG_REJECTED_DEMO_PASSWORD],
};
const tokens = {};
const token = async (user) => {
  assert.ok(users[user], `Неизвестный пользователь в фикстуре: ${user}`);
  tokens[user] ??= await login(...users[user]);
  return tokens[user];
};
const tasks = await fixture('tasks.json');
const files = await fixture('files.json');
for (const user of ['demo', 'second']) {
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
    // Новая задача создаётся в статусе draft; доводим её до нужного статуса по цепочке переходов.
    const chain = {
      todo: ['todo'],
      in_progress: ['todo', 'in_progress'],
      done: ['todo', 'in_progress', 'done'],
    };
    for (const status of chain[item.status ?? 'todo']) {
      await call(`/api/v1/tasks/${task.id}`, auth, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      });
    }
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

// ── Кейс: мероприятия, организации, заявки, обсуждение ─────────────────────
const json = async (response, status, what) => {
  const text = await response.text();
  assert.equal(response.status, status, `${what}: ${response.status} ${text}`);
  return text ? JSON.parse(text) : null;
};
// Админка — отдельный вход: решения модератора идут от суперадминистратора по админской cookie
// (Origin обязателен — gateway проверяет его у админских запросов).
let adminCookie;
const adminLogin = async () => {
  const response = await fetch(`${base}/auth/admin/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Origin: env.APP_ORIGIN },
    body: JSON.stringify({
      email: env.SUPERADMIN_EMAIL,
      password: env.SUPERADMIN_PASSWORD,
    }),
    signal: AbortSignal.timeout(15_000),
  });
  assert.equal(
    response.status,
    200,
    'Не удалось войти в админку суперадминистратором',
  );
  return response.headers
    .get('set-cookie')
    .match(/starter_admin_session=[^;]+/)[0];
};
const as = async (user, path, method = 'GET', body) => {
  const init = {
    method,
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  };
  if (user !== 'admin') return call(path, await token(user), init);
  adminCookie ??= await adminLogin();
  return fetch(`${base}${path}`, {
    ...init,
    headers: {
      Cookie: adminCookie,
      Origin: env.APP_ORIGIN,
      'Content-Type': 'application/json',
    },
    signal: AbortSignal.timeout(15_000),
  });
};
// Время событий — относительно момента сида по Красноярску (UTC+7): демо не устаревает.
const at = (day, time, hours = 0) => {
  const local = new Date(Date.now() + 7 * 3600_000 + day * 86_400_000);
  const [h, m] = time.split(':').map(Number);
  const start =
    Date.UTC(
      local.getUTCFullYear(),
      local.getUTCMonth(),
      local.getUTCDate(),
      h,
      m,
    ) -
    7 * 3600_000;
  return new Date(start + hours * 3600_000).toISOString();
};
const kase = JSON.parse(
  await readFile(new URL('../fixtures/events.json', import.meta.url), 'utf8'),
);
if ((await as('org', '/api/v1/my/organization')).status === 200) {
  console.log('Мероприятия и организации уже есть, пропускаем');
} else {
  const sports = (
    await json(await as('demo', '/api/v1/sports'), 200, 'Справочник')
  ).items;
  const sportID = (name) => sports.find((s) => s.name === name)?.id;
  for (const o of kase.organizations) {
    const org = await json(
      await as(o.user, '/api/v1/my/organization', 'POST', {
        name: o.name,
        description: o.description,
        city: o.city,
        contacts: o.contacts,
        sport_ids: o.sports.map(sportID).filter(Boolean),
      }),
      201,
      `Заявка «${o.name}»`,
    );
    if (o.review) {
      await json(
        await as(
          'admin',
          `/api/v1/admin/organizations/${org.id}/${o.review}`,
          'POST',
          { reason: o.reason ?? '' },
        ),
        200,
        `Решение по «${o.name}»`,
      );
    }
  }
  const events = {};
  for (const e of kase.events) {
    const id = sportID(e.sport);
    const created = await json(
      await as(e.user, '/api/v1/events', 'POST', {
        title: e.title,
        description: e.description,
        ...(id ? { sport_id: id } : { sport_name: e.sport }),
        starts_at: at(e.day, e.time),
        ends_at: at(e.day, e.time, e.hours),
        location: e.location,
        level: e.level,
        requirements: e.requirements,
        price_note: e.price_note,
        private_info: e.private_info,
        join_mode: e.join_mode,
        capacity: e.capacity ?? null,
        as_organization: e.as_organization ?? false,
        lat: e.lat ?? null,
        lng: e.lng ?? null,
        outdoor: e.outdoor ?? false,
        min_participants: e.min_participants ?? null,
      }),
      201,
      `Мероприятие «${e.title}»`,
    );
    events[e.key] = { ...created, user: e.user };
  }
  for (const p of kase.participations) {
    const event = events[p.event];
    const mine = await json(
      await as(p.user, `/api/v1/events/${event.id}/join`, 'POST', {
        message: p.message ?? '',
      }),
      200,
      `Запись ${p.user} на «${event.title}»`,
    );
    if (p.decision) {
      await json(
        await as(
          event.user,
          `/api/v1/events/${event.id}/participants/${mine.id}/${p.decision}`,
          'POST',
          {},
        ),
        200,
        `Решение по заявке на «${event.title}»`,
      );
    }
  }
  for (const c of kase.comments) {
    const event = events[c.event];
    const question = await json(
      await as(c.user, `/api/v1/events/${event.id}/comments`, 'POST', {
        body: c.body,
      }),
      201,
      `Вопрос к «${event.title}»`,
    );
    if (c.reply) {
      await json(
        await as(event.user, `/api/v1/events/${event.id}/comments`, 'POST', {
          body: c.reply,
          parent_id: question.id,
        }),
        201,
        `Ответ в «${event.title}»`,
      );
    }
    if (c.hide) {
      await json(
        await as(
          'admin',
          `/api/v1/admin/comments/${question.id}/hide`,
          'POST',
          { reason: c.hide },
        ),
        200,
        'Скрытие сообщения',
      );
    }
  }
  // Снимаем последним: на снятое событие уже нельзя записаться и писать в обсуждение.
  for (const e of kase.events.filter((e) => e.moderate)) {
    await json(
      await as(
        'admin',
        `/api/v1/admin/events/${events[e.key].id}/unpublish`,
        'POST',
        { reason: e.moderate },
      ),
      200,
      `Снятие «${e.title}»`,
    );
  }
  console.log(
    `Кейс: организаций — ${kase.organizations.length}, мероприятий — ${kase.events.length}, записей — ${kase.participations.length}, сообщений — ${kase.comments.length}`,
  );
}
// «Ищу компанию» — отдельно от мероприятий, чтобы досеять и в уже заполненную базу. Запросы создаются
// после событий: уже опубликованные подходящие сразу попадают в подборку, а новое событие на демо
// присылает уведомление «Нашлась компания».
const company = kase.company_requests ?? [];
const existing = await json(
  await as('demo', '/api/v1/my/company-requests'),
  200,
  'Запросы «Ищу компанию»',
);
if (existing.items.length > 0) {
  console.log('Запросы «Ищу компанию» уже есть, пропускаем');
} else {
  const sports = (
    await json(await as('demo', '/api/v1/sports'), 200, 'Справочник')
  ).items;
  for (const r of company) {
    const id = sports.find((s) => s.name === r.sport)?.id;
    await json(
      await as(r.user, '/api/v1/my/company-requests', 'POST', {
        ...(id ? { sport_id: id } : { sport_name: r.sport }),
        level: r.level,
        day_parts: r.day_parts,
        weekdays: r.weekdays ?? [],
        area: r.area,
        note: r.note,
      }),
      201,
      `Ищу компанию: ${r.user}, «${r.sport}»`,
    );
  }
  console.log(`Запросов «Ищу компанию» — ${company.length}`);
}
// «Соберём состав» и «Встречу новичка» — тоже отдельно, чтобы досеять в заполненную базу.
// Набрано трое из четырёх, Олег готов встречать новичков: на демо Анна жмёт «Записаться, иду впервые» —
// всем приходит «Состав собран», а Анне — «Вас встретит Олег».
const g = kase.gatherings;
if (g) {
  const mine = await json(
    await as(g.event.user, '/api/v1/my/events'),
    200,
    'Мои мероприятия',
  );
  if (mine.items.some((e) => e.title === g.event.title)) {
    console.log('Демо «Соберём состав» уже есть, пропускаем');
  } else {
    const sports = (
      await json(await as('demo', '/api/v1/sports'), 200, 'Справочник')
    ).items;
    const id = sports.find((s) => s.name === g.event.sport)?.id;
    const e = g.event;
    const created = await json(
      await as(e.user, '/api/v1/events', 'POST', {
        title: e.title,
        description: e.description,
        ...(id ? { sport_id: id } : { sport_name: e.sport }),
        starts_at: at(e.day, e.time),
        ends_at: at(e.day, e.time, e.hours),
        location: e.location,
        lat: e.lat,
        lng: e.lng,
        outdoor: e.outdoor,
        level: e.level,
        requirements: e.requirements,
        price_note: e.price_note,
        private_info: e.private_info,
        join_mode: e.join_mode,
        capacity: e.capacity,
        min_participants: e.min_participants,
      }),
      201,
      `Мероприятие «${e.title}»`,
    );
    for (const p of g.participants) {
      await json(
        await as(p.user, `/api/v1/events/${created.id}/join`, 'POST', {
          first_time: p.first_time ?? false,
        }),
        200,
        `Запись ${p.user} на «${e.title}»`,
      );
      if (p.buddy) {
        await json(
          await as(p.user, `/api/v1/events/${created.id}/buddy`, 'POST'),
          200,
          `Проводник ${p.user}`,
        );
      }
    }
    console.log(
      `Демо «Соберём состав»: записано ${g.participants.length} из ${e.min_participants}`,
    );
  }
}
// Админка-аналитика (сервис moderation) строит свои таблицы из доменных событий. Если её база
// новее данных (подключили позже или очистили), переотправляем снимки — повтор безопасен.
{
  const queued = await json(
    await as('admin', '/api/v1/admin/domain/republish', 'POST'),
    202,
    'Переотправка снимков в moderation',
  );
  console.log(`Снимков для админки отправлено: ${queued.queued}`);
}
console.log(
  'Пароли демопользователей — в локальном .env (make demo печатает их).',
);
