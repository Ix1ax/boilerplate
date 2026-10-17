export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

const WEIGHT: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

const SENSITIVE_KEY = /pass|token|authorization|secret|cookie/i;
const MIN_LEVEL: LogLevel = import.meta.env.DEV ? 'debug' : 'warn';

const loggedErrors = new WeakSet<object>();

const redact = (value: unknown, depth = 0): unknown => {
  if (value === null || typeof value !== 'object' || depth > 4) return value;
  if (Array.isArray(value)) return value.map((v) => redact(v, depth + 1));
  return Object.fromEntries(
    Object.entries(value).map(([k, v]) => [
      k,
      SENSITIVE_KEY.test(k) ? '[redacted]' : redact(v, depth + 1),
    ]),
  );
};

// Убираем query-строку: в ней может лежать токен (напр. ?token=... у WS).
const stripUrlQuery = (url: unknown) =>
  typeof url === 'string' ? url.split('?')[0] : url;

// Error сериализуем в безопасный набор полей. Сырой объект в консоль не отдаём:
// у ApiError через `cause` висит axios-ошибка с конфигом запроса (заголовки,
// тело), которую `redact` по context не трогает.
const serializeError = (error: Error): Record<string, unknown> => {
  const e = error as {
    status?: unknown;
    method?: unknown;
    url?: unknown;
    data?: unknown;
  };

  const out: Record<string, unknown> = {
    name: error.name,
    message: error.message,
    stack: error.stack,
  };

  if (e.status !== undefined) out.status = e.status;
  if (e.method !== undefined) out.method = e.method;
  if (e.url !== undefined) out.url = stripUrlQuery(e.url);
  if (e.data !== undefined) out.data = e.data;
  return redact(out) as Record<string, unknown>;
};

const write = (
  level: LogLevel,
  scope: string | undefined,
  message: string,
  details?: unknown,
  extra?: Record<string, unknown>,
) => {
  if (WEIGHT[level] < WEIGHT[MIN_LEVEL]) return;

  let error: Error | undefined;
  let context: Record<string, unknown> | undefined;

  if (details instanceof Error) {
    if (loggedErrors.has(details)) return;
    loggedErrors.add(details);
    error = details;
  } else if (details !== undefined) {
    context =
      typeof details === 'object' && details !== null
        ? (details as Record<string, unknown>)
        : { value: details };
  }
  if (extra) context = { ...context, ...extra };
  if (context) context = redact(context) as Record<string, unknown>;

  const args: unknown[] = [`[${scope ?? 'app'}] ${message}`];
  if (context) args.push(context);
  if (error) args.push(serializeError(error));

  console[level === 'debug' ? 'log' : level](...args);
};

export type Logger = {
  debug: (
    message: string,
    details?: unknown,
    context?: Record<string, unknown>,
  ) => void;
  info: (
    message: string,
    details?: unknown,
    context?: Record<string, unknown>,
  ) => void;
  warn: (
    message: string,
    details?: unknown,
    context?: Record<string, unknown>,
  ) => void;
  error: (
    message: string,
    details?: unknown,
    context?: Record<string, unknown>,
  ) => void;
  child: (scope: string) => Logger;
};

const createLogger = (scope?: string): Logger => ({
  debug: (m, d, c) => write('debug', scope, m, d, c),
  info: (m, d, c) => write('info', scope, m, d, c),
  warn: (m, d, c) => write('warn', scope, m, d, c),
  error: (m, d, c) => write('error', scope, m, d, c),
  child: (s) => createLogger(scope ? `${scope}:${s}` : s),
});

export const logger = createLogger();

let initialized = false;
export const initGlobalErrorHandlers = () => {
  if (initialized) return;
  initialized = true;

  window.addEventListener('error', (e) => {
    logger.error('Необработанная ошибка', e.error ?? e.message, {
      source: e.filename,
      line: e.lineno,
      column: e.colno,
    });
  });

  window.addEventListener('unhandledrejection', (e) => {
    logger.error('Необработанный promise', e.reason);
  });
};
