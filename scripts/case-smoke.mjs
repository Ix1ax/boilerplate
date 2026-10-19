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
};
const as = {};
let petrToken = '';
for (const [name, [email, password]] of Object.entries(accounts)) {
  const token = await login(env[email], env[password]);
  if (name === 'petr') petrToken = token;
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

// Админка — отдельный вход суперадминистратора: своя cookie, Origin приложения у каждого запроса.
const adminLogin = await fetch(`${base}/auth/admin/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json', Origin: env.APP_ORIGIN },
  body: JSON.stringify({
    email: env.SUPERADMIN_EMAIL,
    password: env.SUPERADMIN_PASSWORD,
  }),
  signal: AbortSignal.timeout(15_000),
});
assert.equal(adminLogin.status, 200, 'Не удалось войти в админку');
const adminCookie = adminLogin.headers
  .get('set-cookie')
  .match(/starter_admin_session=[^;]+/)[0];
const adminHeaders = { Cookie: adminCookie, Origin: env.APP_ORIGIN };
as.admin = async (path, method = 'GET', body) => {
  const response = await fetch(`${base}/api/v1${path}`, {
    method,
    headers: { ...adminHeaders, 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(15_000),
  });
  const text = await response.text();
  return { status: response.status, body: text ? JSON.parse(text) : null };
};

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
const companyRequests = [];
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

  // Сверх минимума: «Ищу компанию». Мария ищет компанию — организатор видит спрос — публикует
  // подходящее событие — Марии приходит «Нашлась компания», событие появляется в её подборке.
  const wanted = 'Ультимат-фрисби';
  const before = ok(await as.maria('/my/company-requests'), 'запросы Марии');
  for (const r of before.items.filter(
    (r) => r.sport.name === wanted && r.status === 'active',
  )) {
    ok(
      await as.maria(`/my/company-requests/${r.id}`, 'DELETE'),
      'закрытие старого запроса',
    );
  }
  const wish = ok(
    await as.maria('/my/company-requests', 'POST', {
      sport_name: wanted,
      day_parts: ['morning'],
      note: 'Хочу попробовать',
    }),
    'запрос «Ищу компанию»',
  );
  companyRequests.push(wish.id);
  const demand = ok(
    await as.igor(`/sports/demand?sport_name=${encodeURIComponent(wanted)}`),
    'спрос у организатора',
  );
  assert.ok(
    demand.count >= 1 && demand.by_day_part.morning >= 1,
    'организатор не видит спрос',
  );
  const offer = await createEvent('igor', {
    title: 'Фрисби для новичков',
    sport_name: wanted,
    starts_at: inSaturday(10),
    join_mode: 'open',
  });
  assert.ok(
    offer.company_matches >= 1,
    'событие не подобралось ищущим компанию',
  );
  let invited = false;
  for (let i = 0; i < 20 && !invited; i++) {
    const bell = ok(await as.maria('/notifications'), 'колокольчик Марии');
    invited = bell.items.some(
      (n) =>
        n.title.startsWith('Нашлась компания') &&
        n.link === `/events/${offer.id}`,
    );
    if (!invited) await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  assert.ok(invited, 'уведомление «Нашлась компания» не пришло за 20 секунд');
  const picked = ok(
    await as.maria(`/my/company-requests/${wish.id}/matches`),
    'подборка Марии',
  );
  assert.ok(
    picked.items.some((e) => e.id === offer.id),
    'событие не попало в подборку',
  );
  step(
    '★ Ищу компанию: спрос виден организатору, новое событие пришло уведомлением и попало в подборку',
  );

  // Сверх минимума: «Соберём состав», «Встречу новичка», напоминание и погода.
  const squad = await createEvent('anna', {
    title: 'Волейбол на пляже',
    sport_name: 'Волейбол',
    join_mode: 'open',
    outdoor: true,
    lat: 56.0285,
    lng: 92.9271,
    min_participants: 2,
    private_info: 'Сетка у спасательной вышки',
  });
  assert.equal(squad.quorum_status, 'gathering');
  ok(await as.igor(`/events/${squad.id}/join`, 'POST', {}), 'запись Игоря');
  ok(await as.igor(`/events/${squad.id}/buddy`, 'POST'), 'Игорь — проводник');
  const newcomer = ok(
    await as.petr(`/events/${squad.id}/join`, 'POST', { first_time: true }),
    'запись новичка',
  );
  assert.ok(newcomer.first_time);
  const petrCard = ok(await as.petr(`/events/${squad.id}`), 'карточка новичка');
  assert.equal(
    petrCard.quorum_status,
    'reached',
    'состав не собран после второго',
  );
  assert.equal(
    petrCard.my_participation.buddy?.name?.startsWith('Игорь'),
    true,
    'новичку не назначен проводник',
  );
  const reminded = ok(
    await as.anna(`/events/${squad.id}/remind`, 'POST'),
    'напоминание',
  );
  assert.equal(reminded.sent, 2);
  assert.equal(
    (await as.anna(`/events/${squad.id}/remind`, 'POST')).status,
    409,
    'напоминание чаще раза в час',
  );
  const forecast = ok(await as.petr(`/events/${squad.id}/weather`), 'погода');
  assert.equal(typeof forecast.available, 'boolean');
  step(
    `★ Соберём состав и встречу новичка: состав собран, проводник назначен, напоминание ушло; погода — ${forecast.available ? forecast.forecast.description : forecast.reason}`,
  );

  // Для фронта: профиль, правка и перенос события, поиск рядом, другая встреча, календарь.
  const me = ok(
    await as.anna('/my/profile', 'PATCH', {
      about: 'Люблю утренние пробежки',
      level: 'beginner',
    }),
    'правка профиля',
  );
  assert.equal(me.level, 'beginner');
  assert.ok(me.stats, 'нет статистики организатора в профиле');
  const later = new Date(Date.parse(own.starts_at) + 3600_000).toISOString();
  const moved = ok(
    await as.anna(`/events/${own.id}`, 'PATCH', {
      starts_at: later,
      location: 'Красноярск, остров Татышев',
    }),
    'перенос события',
  );
  assert.equal(Date.parse(moved.starts_at), Date.parse(later));
  assert.equal(
    (await as.igor(`/events/${own.id}`, 'PATCH', { title: 'Чужое' })).status,
    403,
    'чужое событие изменено',
  );
  let warned = false;
  for (let i = 0; i < 20 && !warned; i++) {
    const bell = ok(await as.igor('/notifications'), 'колокольчик Игоря');
    warned = bell.items.some(
      (n) => n.title.startsWith('Перенос:') && n.link === `/events/${own.id}`,
    );
    if (!warned) await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  assert.ok(warned, 'участник не узнал о переносе');
  const nearby = ok(
    await as.petr(`/events?near=56.0285,92.9271&radius_km=2`),
    'события рядом',
  );
  assert.ok(
    nearby.items.some(
      (e) => e.id === squad.id && typeof e.distance_km === 'number',
    ),
    'нет события рядом',
  );
  ok(await as.petr(`/events/${squad.id}/alternatives`), 'другие встречи');
  const ics = await fetch(`${base}/api/v1/events/${squad.id}/calendar.ics`, {
    headers: { Authorization: `Bearer ${petrToken}` },
  });
  assert.ok((await ics.text()).startsWith('BEGIN:VCALENDAR'), 'файл календаря');
  step(
    '★ Для фронта: профиль, перенос с уведомлением, поиск рядом, другие встречи, календарь',
  );

  // Админка: карточки и история в events, аналитика и журнал — в отдельном сервисе moderation.
  const orgCard = ok(
    await as.admin(`/admin/organizations/${ladya.id}`),
    'карточка организации',
  );
  assert.ok(
    orgCard.history.some((h) => h.to === 'approved'),
    'в карточке нет истории допуска',
  );
  assert.ok(
    orgCard.events.some((e) => e.id === orgEvent.id),
    'в карточке нет событий организации',
  );
  const takedownHistory = ok(
    await as.admin(`/admin/events/${custom.id}/history`),
    'история события',
  );
  assert.ok(
    takedownHistory.items.some(
      (h) => h.to === 'unpublished' && h.comment === takedown,
    ),
  );
  const sportsAdmin = ok(
    await as.admin('/admin/sports'),
    'справочник для модерации',
  );
  const frisbee = sportsAdmin.items.find((x) => x.name === 'Ультимат-фрисби');
  assert.ok(frisbee?.is_custom, 'нет пользовательского направления');
  ok(
    await as.admin(`/admin/sports/${frisbee.id}/hide`, 'POST', {
      reason: 'Проверка модерации',
    }),
    'скрытие направления',
  );
  const visibleSports = ok(
    await as.anna('/sports'),
    'справочник после скрытия',
  );
  assert.ok(
    !visibleSports.items.some((x) => x.id === frisbee.id),
    'скрытое направление в справочнике',
  );
  ok(
    await as.admin(`/admin/sports/${frisbee.id}/show`, 'POST'),
    'возврат направления',
  );
  assert.equal(
    (await as.anna('/admin/moderation/dashboard')).status,
    401,
    'дашборд доступен без админской сессии',
  );
  let logged = false;
  for (let i = 0; i < 20 && !logged; i++) {
    const journal = ok(
      await as.admin(`/admin/moderation/log?entity_id=${custom.id}`),
      'журнал модерации',
    );
    logged = journal.items.some(
      (e) => e.to === 'unpublished' && e.comment === takedown,
    );
    if (!logged) await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  assert.ok(logged, 'снятие не попало в журнал модерации за 20 секунд');
  const board = ok(await as.admin('/admin/moderation/dashboard'), 'дашборд');
  assert.ok(
    board.events.unpublished >= 1 && board.series.length === 14,
    'дашборд пустой',
  );
  const csv = await fetch(
    `${base}/api/v1/admin/moderation/reports/events.csv`,
    {
      headers: adminHeaders,
    },
  );
  assert.equal(csv.status, 200);
  // text() снимает BOM при декодировании, поэтому проверяем байты: Excel нужен UTF-8 с BOM.
  const bytes = new Uint8Array(await csv.arrayBuffer());
  assert.deepEqual(
    [...bytes.slice(0, 3)],
    [0xef, 0xbb, 0xbf],
    'CSV без BOM — Excel покажет кракозябры',
  );
  assert.ok(
    new TextDecoder().decode(bytes).startsWith('Название;'),
    'CSV не для Excel',
  );
  step(
    '★ Админка: карточки с историей, модерация направлений, журнал и дашборд из moderation, CSV',
  );

  // Жалобы, исключение участника, удаление и возврат сообщения.
  {
    const meet = await createEvent('igor', {
      title: 'Воркаут на турниках',
      sport_id: running.id,
      join_mode: 'open',
      capacity: 1,
    });
    const annaJoin = ok(
      await as.anna(`/events/${meet.id}/join`, 'POST'),
      'запись Анны',
    );
    const annaPid = annaJoin.id;
    const petrJoin = ok(
      await as.petr(`/events/${meet.id}/join`, 'POST'),
      'запись Петра',
    );
    assert.equal(
      petrJoin.status,
      'waitlist',
      'второй на одно место — в лист ожидания',
    );
    const spam = ok(
      await as.anna(`/events/${meet.id}/comments`, 'POST', {
        body: 'Продам протеин, пишите в личку',
      }),
      'сообщение Анны',
    );
    assert.equal(
      (
        await as.anna(`/events/${meet.id}/comments/${spam.id}/report`, 'POST', {
          reason: 'spam',
        })
      ).status,
      409,
      'на своё сообщение жаловаться нельзя',
    );
    ok(
      await as.maria(`/events/${meet.id}/comments/${spam.id}/report`, 'POST', {
        reason: 'spam',
      }),
      'жалоба Марии',
    );
    assert.equal(
      (
        await as.maria(
          `/events/${meet.id}/comments/${spam.id}/report`,
          'POST',
          { reason: 'spam' },
        )
      ).status,
      409,
      'повторная жалоба не создаёт дубль',
    );
    const queue = ok(
      await as.admin(
        '/admin/reports?status=open&target_type=comment&limit=100',
      ),
      'очередь жалоб',
    );
    const report = queue.items.find((r) => r.target_id === spam.id);
    assert.ok(
      report && report.target.text.includes('протеин'),
      'админ видит текст сообщения в жалобе',
    );
    // Подсказка ИИ только советует: без Qwen остаются правила (source: rules), решает админ.
    const hint = ok(
      await as.admin(`/admin/reports/${report.id}/analyze`, 'POST'),
      'подсказка по жалобе',
    );
    assert.ok(
      hint.decision_required &&
        hint.accept.path === `/api/v1/admin/comments/${spam.id}/hide`,
      'подсказка оставляет решение админу и называет его действие',
    );
    ok(
      await as.admin(`/admin/comments/${spam.id}/hide`, 'POST', {
        reason: 'Реклама',
      }),
      'скрытие',
    );
    const mine = ok(await as.maria('/my/reports'), 'мои жалобы');
    assert.equal(
      mine.items.find((r) => r.target_id === spam.id)?.status,
      'resolved',
      'жалоба закрыта скрытием',
    );
    ok(
      await as.admin(`/admin/comments/${spam.id}/restore`, 'POST'),
      'возврат сообщения',
    );
    ok(
      await as.anna(`/events/${meet.id}/comments/${spam.id}`, 'DELETE'),
      'автор удалил сообщение',
    );
    const thread = ok(
      await as.maria(`/events/${meet.id}/comments`),
      'обсуждение',
    );
    const gone = thread.items.find((c) => c.id === spam.id);
    assert.ok(
      gone.status === 'deleted' && gone.body === '',
      'текст удалённого не виден',
    );
    ok(
      await as.igor(
        `/events/${meet.id}/participants/${annaPid}/remove`,
        'POST',
        { reason: 'Не по теме встречи' },
      ),
      'исключение участника',
    );
    const petrView = ok(
      await as.petr(`/events/${meet.id}`),
      'карточка у Петра',
    );
    assert.equal(
      petrView.my_participation?.status,
      'confirmed',
      'место перешло первому из листа ожидания',
    );
    assert.equal(
      (await as.anna(`/events/${meet.id}/join`, 'POST')).status,
      409,
      'исключённый не записывается снова',
    );
    const eventReport = ok(
      await as.maria(`/events/${meet.id}/report`, 'POST', {
        reason: 'wrong_info',
        details: 'Турников там нет',
      }),
      'жалоба на событие',
    );
    ok(
      await as.admin(`/admin/reports/${eventReport.id}/dismiss`, 'POST', {
        resolution: 'Площадка на месте',
      }),
      'отказ по жалобе',
    );
    // Отказ доходит до moderation доменным событием: журнал, дашборд и CSV жалоб. Проекция отстаёт на секунды.
    let reportLogged = false;
    for (let i = 0; i < 20 && !reportLogged; i++) {
      const journal = ok(
        await as.admin(
          `/admin/moderation/log?entity=report&entity_id=${eventReport.id}`,
        ),
        'журнал жалоб',
      );
      reportLogged = journal.items.some(
        (e) => e.to === 'dismissed' && e.comment === 'Площадка на месте',
      );
      if (!reportLogged)
        await new Promise((resolve) => setTimeout(resolve, 1000));
    }
    assert.ok(
      reportLogged,
      'отказ по жалобе не попал в журнал модерации за 20 секунд',
    );
    const reportBoard = ok(
      await as.admin('/admin/moderation/dashboard'),
      'дашборд с жалобами',
    );
    assert.ok(
      reportBoard.reports.dismissed_week >= 1 && reportBoard.sports.custom >= 1,
      'на дашборде нет жалоб или пользовательских направлений',
    );
    const reportsCsv = await fetch(
      `${base}/api/v1/admin/moderation/reports/reports.csv`,
      { headers: adminHeaders },
    );
    assert.equal(reportsCsv.status, 200);
    assert.ok(
      (await reportsCsv.text()).includes('Площадка на месте'),
      'отказ по жалобе не попал в CSV',
    );
    step(
      '★ Модерация: жалобы, удаление и возврат сообщения, исключение участника с передачей места, журнал и CSV жалоб',
    );
  }

  console.log(
    'Все шесть сценариев кейса, дополнительные возможности и админка пройдены.',
  );
} finally {
  for (const id of companyRequests) {
    await as.maria(`/my/company-requests/${id}`, 'DELETE');
  }
  // Убираем за собой: созданные события удаляем с причиной, чтобы они не мешали демо.
  for (const id of created) {
    await as.admin(`/admin/events/${id}/delete`, 'POST', {
      reason: 'Служебная проверка',
    });
  }
}
