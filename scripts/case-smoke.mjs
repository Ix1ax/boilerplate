// Сквозная проверка шести базовых сценариев кейса на живом стеке через API — от лица демо-пользователей,
// с проверкой результата у второй стороны (организатор видит участника, организация — решение админа и т. д.).
// Сид-данные не портит: созданные события в конце удаляются админом, заявка «Сосен» возвращается в ожидание.
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
const run = Date.now().toString(36);

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

const accounts = {
  anna: ['DEMO_EMAIL', 'DEMO_PASSWORD'],
  igor: ['SECOND_DEMO_EMAIL', 'SECOND_DEMO_PASSWORD'],
  oleg: ['ORG_DEMO_EMAIL', 'ORG_DEMO_PASSWORD'],
  maria: ['ORG_PENDING_DEMO_EMAIL', 'ORG_PENDING_DEMO_PASSWORD'],
  petr: ['ORG_REJECTED_DEMO_EMAIL', 'ORG_REJECTED_DEMO_PASSWORD'],
  admin: ['ADMIN_DEMO_EMAIL', 'ADMIN_DEMO_PASSWORD'],
};
const as = {};
for (const [name, [email, password]] of Object.entries(accounts)) {
  const token = await login(env[email], env[password]);
  as[name] = async (path, method = 'GET', body) => {
    const response = await fetch(`${base}/api/v1${path}`, {
      method,
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: AbortSignal.timeout(15_000),
    });
    const text = await response.text();
    return { status: response.status, body: text ? JSON.parse(text) : null };
  };
}

const ok = (res, what) => {
  assert.ok(
    res.status >= 200 && res.status < 300,
    `${what}: ${res.status} ${JSON.stringify(res.body)}`,
  );
  return res.body;
};
const inSaturday = (hour) => {
  const d = new Date(Date.now() + 5 * 24 * 3600 * 1000);
  d.setUTCHours(hour - 7, 0, 0, 0); // Красноярск — UTC+7
  return d.toISOString();
};
const created = [];
const createEvent = async (who, input) => {
  const event = ok(
    await as[who]('/events', 'POST', {
      location: 'Красноярск, остров Татышев, у главного входа',
      starts_at: inSaturday(10),
      level: 'any',
      ...input,
      title: `${input.title} · проверка ${run}`,
    }),
    `создание события «${input.title}»`,
  );
  created.push(event.id);
  return event;
};
const step = (name) => console.log(`✓ ${name}`);

try {
  // Сценарий 1. Создание мероприятия пользователем, в том числе со своим направлением.
  const sports = ok(await as.anna('/sports'), 'справочник направлений');
  const running = (sports.items ?? sports).find((s) => s.name === 'Бег');
  assert.ok(running, 'в справочнике нет направления «Бег»');
  const own = await createEvent('anna', {
    title: 'Утренняя пробежка',
    sport_id: running.id,
    requirements: 'Удобная обувь, вода',
    join_mode: 'open',
    capacity: 10,
  });
  assert.equal(own.organizer.type, 'user');
  const custom = await createEvent('anna', {
    title: 'Ультимат-фрисби на траве',
    sport_name: 'Ультимат-фрисби',
    join_mode: 'open',
  });
  assert.equal(custom.sport.name, 'Ультимат-фрисби');
  assert.ok(
    custom.sport.is_custom,
    'своё направление не помечено как пользовательское',
  );
  const found = ok(
    await as.igor(`/events?q=${encodeURIComponent(run)}`),
    'поиск под другим пользователем',
  );
  assert.ok(
    found.items.some((e) => e.id === own.id),
    'Игорь не находит событие Анны',
  );
  ok(
    await as.igor(`/events/${custom.id}`),
    'карточка под другим пользователем',
  );
  step(
    '1. Пользователь создаёт событие, в том числе со своим направлением; другой его находит',
  );

  // Сценарий 2. Допуск организации: отказ с причиной и повторная подача, публикация допущенной.
  const pending = ok(await as.maria('/my/organization'), 'заявка «Сосен»');
  assert.equal(pending.status, 'pending');
  const denied = await as.maria('/events', 'POST', {
    title: 'Ходьба от «Сосен»',
    sport_name: 'Скандинавская ходьба',
    location: 'Красноярск, Центральный парк',
    starts_at: inSaturday(9),
    join_mode: 'open',
    as_organization: true,
  });
  assert.equal(
    denied.status,
    403,
    'недопущенная организация смогла опубликовать событие',
  );
  const queue = ok(
    await as.admin('/admin/organizations?status=pending'),
    'очередь заявок',
  );
  assert.ok(
    queue.items.some((o) => o.id === pending.id),
    'админ не видит заявку «Сосен»',
  );
  const reason = 'Укажите расписание и контакты тренера';
  ok(
    await as.admin(`/admin/organizations/${pending.id}/reject`, 'POST', {
      reason,
    }),
    'отклонение заявки',
  );
  const rejected = ok(
    await as.maria('/my/organization'),
    'результат для «Сосен»',
  );
  assert.equal(rejected.status, 'rejected');
  assert.equal(rejected.review_reason, reason);
  const resubmitted = ok(
    await as.maria('/my/organization', 'PUT', {
      name: rejected.name,
      description: rejected.description,
      city: rejected.city,
      contacts: rejected.contacts,
      sport_ids: rejected.sports.map((s) => s.id),
    }),
    'повторная подача',
  );
  assert.equal(resubmitted.status, 'pending');
  const ladya = ok(await as.oleg('/my/organization'), 'организация «Ладья»');
  assert.equal(ladya.status, 'approved');
  const orgEvent = await createEvent('oleg', {
    title: 'Блиц-турнир',
    sport_name: 'Шахматы',
    join_mode: 'open',
    as_organization: true,
  });
  assert.equal(orgEvent.organizer.type, 'organization');
  assert.equal(
    orgEvent.status,
    'published',
    'событие организации ушло на модерацию',
  );
  const page = ok(
    await as.igor(`/organizations/${ladya.id}`),
    'страница «Ладьи»',
  );
  const pageEvents = page.events ?? page.upcoming_events ?? [];
  assert.ok(
    pageEvents.some((e) => e.id === orgEvent.id),
    'на странице «Ладьи» нет нового события',
  );
  step(
    '2. Админ отклоняет заявку с причиной, организация её видит; допущенная публикует без модерации',
  );

  // Сценарий 3. Свободное присоединение.
  const joined = ok(
    await as.igor(`/events/${own.id}/join`, 'POST', {}),
    'запись Игоря',
  );
  assert.equal(joined.status, 'confirmed');
  const mine = ok(await as.igor('/my/participations'), 'записи Игоря');
  assert.ok(
    mine.items.some((p) => p.event.id === own.id && p.status === 'confirmed'),
  );
  const roster = ok(
    await as.anna(`/events/${own.id}/participants`),
    'участники у Анны',
  );
  assert.ok(
    (roster.items ?? roster).some(
      (p) => p.status === 'confirmed' && p.user.name.startsWith('Игорь'),
    ),
    'организатор не видит Игоря среди участников',
  );
  step('3. Свободная запись сразу подтверждена, организатор видит участника');

  // Сценарий 4. Участие по заявке: принятие и отклонение.
  const byRequest = await createEvent('anna', {
    title: 'Йога для начинающих',
    sport_name: 'Йога',
    join_mode: 'request',
    private_info: 'Вход со двора, код домофона 24',
  });
  const igorApp = ok(
    await as.igor(`/events/${byRequest.id}/join`, 'POST', {
      message: 'Занимаюсь полгода',
    }),
    'заявка Игоря',
  );
  assert.equal(igorApp.status, 'pending');
  const hidden = ok(
    await as.igor(`/events/${byRequest.id}`),
    'карточка до решения',
  );
  assert.equal(
    hidden.private_info,
    '',
    'закрытая информация видна до подтверждения',
  );
  const petrApp = ok(
    await as.petr(`/events/${byRequest.id}/join`, 'POST', {}),
    'заявка Петра',
  );
  ok(
    await as.anna(
      `/events/${byRequest.id}/participants/${igorApp.id}/approve`,
      'POST',
      {},
    ),
    'принятие заявки',
  );
  ok(
    await as.anna(
      `/events/${byRequest.id}/participants/${petrApp.id}/reject`,
      'POST',
      {
        reason: 'Группа только для начинающих',
      },
    ),
    'отклонение заявки',
  );
  const admitted = ok(
    await as.igor(`/events/${byRequest.id}`),
    'карточка после принятия',
  );
  assert.equal(admitted.my_participation.status, 'confirmed');
  assert.ok(
    admitted.private_info.includes('домофона'),
    'после принятия не видна закрытая информация',
  );
  const refused = ok(
    await as.petr(`/events/${byRequest.id}`),
    'карточка после отказа',
  );
  assert.equal(refused.my_participation.status, 'rejected');
  const forbidden = await as.admin(
    `/events/${byRequest.id}/participants/${petrApp.id}/approve`,
    'POST',
    {},
  );
  assert.ok(
    forbidden.status >= 400,
    'админ смог решить заявку вместо организатора',
  );
  step(
    '4. Заявка ждёт решения; организатор принимает и отклоняет, статус виден обеим сторонам',
  );

  // Сценарий 5. Вопрос в обсуждении и ответ организатора.
  const question = ok(
    await as.igor(`/events/${own.id}/comments`, 'POST', {
      body: 'Какой темп пробежки?',
    }),
    'вопрос',
  );
  ok(
    await as.anna(`/events/${own.id}/comments`, 'POST', {
      body: 'Спокойный, 6:30 на километр',
      parent_id: question.id,
    }),
    'ответ',
  );
  const thread = ok(
    await as.petr(`/events/${own.id}/comments`),
    'обсуждение под третьим пользователем',
  );
  const answer = thread.items.find((c) => c.parent_id === question.id);
  assert.ok(answer?.is_organizer, 'ответ организатора не отмечен');
  assert.ok(!thread.items.find((c) => c.id === question.id).is_organizer);
  ok(
    await as.admin(`/admin/comments/${question.id}/hide`, 'POST', {
      reason: 'Проверка модерации',
    }),
    'скрытие сообщения',
  );
  const moderated = ok(
    await as.petr(`/events/${own.id}/comments`),
    'обсуждение после скрытия',
  );
  const hiddenOne = moderated.items.find((c) => c.id === question.id);
  assert.ok(
    !hiddenOne || hiddenOne.status === 'hidden',
    'скрытое сообщение осталось видимым',
  );
  assert.ok(
    moderated.items.some((c) => c.id === answer.id),
    'скрытие задело остальное обсуждение',
  );
  step(
    '5. Вопрос и ответ различимы по авторству; админ скрывает одно сообщение',
  );

  // Сценарий 6. Снятие с публикации: нет в выдаче, нельзя записаться по прямой ссылке.
  ok(
    await as.igor(`/events/${custom.id}/join`, 'POST', {}),
    'запись до снятия',
  );
  const takedown = 'Неверно указано место проведения';
  ok(
    await as.admin(`/admin/events/${custom.id}/unpublish`, 'POST', {
      reason: takedown,
    }),
    'снятие с публикации',
  );
  const feed = ok(
    await as.petr(`/events?q=${encodeURIComponent(run)}`),
    'выдача после снятия',
  );
  assert.ok(
    !feed.items.some((e) => e.id === custom.id),
    'снятое событие осталось в выдаче',
  );
  const direct = await as.petr(`/events/${custom.id}`);
  assert.equal(direct.status, 404, 'постороннему открывается снятое событие');
  const sneak = await as.petr(`/events/${custom.id}/join`, 'POST', {});
  assert.ok(
    sneak.status >= 400,
    'на снятое событие удалось записаться по прямой ссылке',
  );
  const forMember = ok(
    await as.igor(`/events/${custom.id}`),
    'снятое событие у участника',
  );
  assert.equal(forMember.status, 'unpublished');
  assert.equal(forMember.moderation_reason, takedown);
  const forOrganizer = ok(
    await as.anna('/my/events'),
    'мои события у организатора',
  );
  const mineTaken = forOrganizer.items.find((e) => e.id === custom.id);
  assert.equal(
    mineTaken?.status,
    'unpublished',
    'организатор не видит, что событие снято',
  );
  step(
    '6. Снятое событие пропало из выдачи, записаться нельзя; участник и организатор видят причину',
  );

  console.log('Все шесть сценариев кейса пройдены.');
} finally {
  // Убираем за собой: созданные события удаляем с причиной, чтобы они не мешали демо.
  for (const id of created) {
    await as.admin(`/admin/events/${id}/delete`, 'POST', {
      reason: 'Служебная проверка',
    });
  }
}
