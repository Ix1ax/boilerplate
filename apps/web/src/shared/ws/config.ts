// Инъекция зависимостей для WS-слоя по образцу configureApi из shared/api.
// shared/ws не знает про entities — токен и колбэки прокидываются из app-слоя.

export type WsConfig = {
  /** Базовый URL сокета, напр. wss://host/ws (без token). */
  getUrl: () => string;
  /** Текущий access-токен; подставляется в query при каждом (пере)подключении. */
  getToken: () => string | null;
  /** Вызывается, когда сервер закрывает сокет по причине авторизации. */
  onUnauthorized: () => void;
};

let current: WsConfig = {
  getUrl: () => '',
  getToken: () => null,
  onUnauthorized: () => {},
};

export const configureWs = (next: Partial<WsConfig>) => {
  current = { ...current, ...next };
};

export const getWsConfig = () => current;

/** Настройки reconnect/heartbeat. Правятся в одном месте. */
export const WS_TUNING = {
  reconnect: {
    baseDelay: 1_000,
    maxDelay: 30_000,
    factor: 2,
    /** Доля случайного разброса задержки (±20%). */
    jitter: 0.2,
  },
  heartbeat: {
    /** Интервал отправки ping. */
    interval: 25_000,
    /** Сколько ждём pong до принудительного reconnect. */
    pongTimeout: 10_000,
  },
  queue: {
    /** Максимум сообщений в очереди до открытия (старые вытесняются). */
    maxSize: 100,
    /** Срок жизни сообщения в очереди; протухшие не отправляются. */
    ttl: 30_000,
  },
} as const;
