// Нагрузочный тест: чтение списка (основная нагрузка) и запись под лимитом gateway.
// Запуск: make loadtest (LOAD_VUS, LOAD_DURATION — необязательно). Идёт по внутренней сети Compose.
import http from 'k6/http';
import { check } from 'k6';

const GATEWAY = 'http://gateway-service:8080';
// Токен из внутреннего адреса Keycloak: issuer в нём публичный (KC_HOSTNAME), gateway его принимает.
const TOKEN_URL =
  'http://keycloak:8080/realms/hackathon/protocol/openid-connect/token';

export const options = {
  summaryTrendStats: ['avg', 'med', 'p(95)', 'p(99)', 'max'],
  scenarios: {
    // Чтение без пауз: показывает предельную пропускную способность стека на этой машине.
    read: {
      executor: 'constant-vus',
      vus: Number(__ENV.LOAD_VUS || 20),
      duration: __ENV.LOAD_DURATION || '30s',
      exec: 'read',
    },
    // Запись: gateway ограничивает 60 изменений в минуту на пользователя, поэтому темп ниже лимита.
    write: {
      executor: 'constant-arrival-rate',
      rate: 40,
      timeUnit: '1m',
      duration: __ENV.LOAD_DURATION || '30s',
      preAllocatedVUs: 2,
      exec: 'write',
    },
  },
  thresholds: {
    'http_req_failed{scenario:read}': ['rate<0.01'],
    'http_req_duration{scenario:read}': ['p(95)<300'],
    'http_req_duration{scenario:write}': ['p(95)<800'],
  },
};

export function setup() {
  const res = http.post(TOKEN_URL, {
    grant_type: 'password',
    client_id: 'hackathon-cli',
    username: __ENV.DEMO_EMAIL,
    password: __ENV.DEMO_PASSWORD,
  });
  check(res, { 'вход выполнен': (r) => r.status === 200 });
  return {
    headers: {
      Authorization: `Bearer ${res.json('access_token')}`,
      'Content-Type': 'application/json',
    },
  };
}

export function read({ headers }) {
  const res = http.get(`${GATEWAY}/api/v1/tasks?limit=20`, {
    headers,
    tags: { name: 'GET /tasks' },
  });
  check(res, { 'список 200': (r) => r.status === 200 });
}

export function write({ headers }) {
  const created = http.post(
    `${GATEWAY}/api/v1/tasks`,
    JSON.stringify({ title: `Нагрузка ${__VU}-${__ITER}` }),
    {
      headers,
      tags: { name: 'POST /tasks' },
    },
  );
  check(created, { 'создание 201': (r) => r.status === 201 });
  if (created.status === 201) {
    http.del(`${GATEWAY}/api/v1/tasks/${created.json('id')}`, null, {
      headers,
      tags: { name: 'DELETE /tasks/:id' },
    });
  }
}

// Короткая сводка в консоль и JSON для слайда «Масштабируемость».
export function handleSummary(data) {
  const m = data.metrics;
  const read = m['http_req_duration{scenario:read}'];
  const summary = {
    rps: Math.round(m.http_reqs.values.rate),
    requests: m.http_reqs.values.count,
    failed_rate: m.http_req_failed.values.rate,
    read_p95_ms: read ? Math.round(read.values['p(95)']) : null,
    read_p99_ms: read ? Math.round(read.values['p(99)'] ?? 0) : null,
    thresholds_ok: Object.values(m).every(
      (x) => !x.thresholds || Object.values(x.thresholds).every((t) => t.ok),
    ),
  };
  return {
    stdout: `\nRPS: ${summary.rps}, запросов: ${summary.requests}, ошибок: ${(summary.failed_rate * 100).toFixed(2)}%, чтение p95: ${summary.read_p95_ms} мс, пороги: ${summary.thresholds_ok ? 'пройдены' : 'НЕ пройдены'}\n`,
    '/scripts/result.json': JSON.stringify(summary, null, 2),
  };
}
