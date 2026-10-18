import { logger } from '@/shared/lib';

import { getWsConfig, WS_TUNING } from './config';
import type { WsHandler, WsIncoming, WsOutgoing, WsStatus } from './types';
import { wsIncomingSchema } from './types';

const log = logger.child('ws');

const NORMAL_CLOSURE = 1000;
/** Коды закрытия, означающие проблему авторизации — реконнект не поможет. */
const AUTH_CLOSE_CODES = new Set([1008, 4001]);

type StatusListener = (status: WsStatus) => void;
type AnyHandler = (message: WsIncoming) => void;

/** Сообщение в очереди отправки + момент постановки (для TTL). */
type QueuedMessage = { raw: string; ts: number };

/**
 * Единый клиент нативного WebSocket: авто-reconnect с backoff+jitter,
 * heartbeat (ping/pong), очередь исходящих до открытия, типизированный pub/sub.
 */
export class WsClient {
  private socket: WebSocket | null = null;
  private status: WsStatus = 'idle';
  private manualClose = false;
  private attempt = 0;

  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private pongTimer: ReturnType<typeof setTimeout> | null = null;

  private queue: QueuedMessage[] = [];
  private handlers = new Map<string, Set<AnyHandler>>();
  private statusListeners = new Set<StatusListener>();
  /** Каналы активных подписок — пересылаем `subscribe` после reconnect. */
  private channels = new Set<string>();

  getStatus() {
    return this.status;
  }

  /** Подключиться (идемпотентно). */
  connect() {
    this.manualClose = false;
    if (this.isLive()) return;
    this.bindWindowEvents();
    this.open();
  }

  /** Закрыть вручную — без последующего reconnect. Чистит очередь и подписки. */
  close() {
    this.manualClose = true;
    this.clearTimers();
    this.unbindWindowEvents();
    this.teardownSocket();
    this.queue = [];
    this.channels.clear();
    this.setStatus('closed');
  }

  /**
   * Пересоздать соединение с нуля — например при смене сессии (новый токен).
   * Сбрасывает очередь/подписки и открывает свежий сокет с актуальным токеном.
   */
  reconnect() {
    this.manualClose = false;
    this.clearTimers();
    this.teardownSocket();
    this.queue = [];
    this.channels.clear();
    this.attempt = 0;
    this.bindWindowEvents();
    this.open();
  }

  /** Отправить сообщение; если сокет не открыт — встаёт в очередь. */
  send(message: WsOutgoing) {
    this.trackSubscription(message);
    const raw = JSON.stringify(message);
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(raw);
    } else {
      this.enqueue(raw);
    }
  }

  /** Подписка на тип входящего сообщения. Возвращает функцию отписки. */
  on<T extends WsIncoming['type']>(type: T, handler: WsHandler<T>) {
    let set = this.handlers.get(type);
    if (!set) {
      set = new Set();
      this.handlers.set(type, set);
    }
    set.add(handler as AnyHandler);
    return () => {
      const current = this.handlers.get(type);
      if (!current) return;
      current.delete(handler as AnyHandler);
      if (current.size === 0) this.handlers.delete(type);
    };
  }

  /** Подписка на смену статуса соединения. */
  onStatus(listener: StatusListener) {
    this.statusListeners.add(listener);
    return () => {
      this.statusListeners.delete(listener);
    };
  }

  // --- внутреннее ---

  private isLive() {
    const state = this.socket?.readyState;
    return state === WebSocket.OPEN || state === WebSocket.CONNECTING;
  }

  private buildUrl() {
    const { getUrl, getToken } = getWsConfig();
    const base = getUrl();
    if (!base) return '';
    const token = getToken();
    if (!token) return base;
    const sep = base.includes('?') ? '&' : '?';
    return `${base}${sep}token=${encodeURIComponent(token)}`;
  }

  private open() {
    const url = this.buildUrl();
    if (!url) {
      log.warn('WS url не задан — подключение пропущено');
      return;
    }
    this.setStatus(this.attempt === 0 ? 'connecting' : 'reconnecting');

    const socket = new WebSocket(url);
    this.socket = socket;
    // Привязываем обработчики к конкретному экземпляру: запоздалое событие
    // старого сокета (напр. connect→close→connect в StrictMode) не должно
    // трогать уже актуальный this.socket.
    socket.onopen = () => this.onOpen(socket);
    socket.onmessage = (event) => this.onMessage(socket, event);
    socket.onclose = (event) => this.onClose(socket, event);
    socket.onerror = () => {
      if (this.socket === socket) log.warn('WS ошибка соединения');
    };
  }

  private onOpen(socket: WebSocket) {
    if (socket !== this.socket) return;
    this.attempt = 0;
    this.setStatus('open');
    log.info('WS подключён');
    this.flushQueue();
    this.resubscribe();
    this.startHeartbeat();
  }

  private onMessage(socket: WebSocket, event: MessageEvent) {
    if (socket !== this.socket) return;
    // любое сообщение — свидетельство живого соединения
    this.clearPongTimer();

    let raw: unknown;
    try {
      raw = JSON.parse(event.data as string);
    } catch {
      log.warn('WS невалидный JSON', { raw: event.data });
      return;
    }

    const parsed = wsIncomingSchema.safeParse(raw);
    if (!parsed.success) {
      log.warn('WS сообщение не прошло валидацию', { raw: event.data });
      return;
    }

    if (parsed.data.type === 'pong') return;
    this.emit(parsed.data);
  }

  private emit(message: WsIncoming) {
    const set = this.handlers.get(message.type);
    if (!set) return;
    for (const handler of set) {
      try {
        handler(message);
      } catch (error) {
        log.error('WS обработчик упал', error, { type: message.type });
      }
    }
  }

  private onClose(socket: WebSocket, event: CloseEvent) {
    if (socket !== this.socket) return;
    this.clearTimers();
    this.socket = null;

    if (AUTH_CLOSE_CODES.has(event.code)) {
      log.warn('WS закрыт из-за авторизации', { code: event.code });
      this.setStatus('closed');
      getWsConfig().onUnauthorized();
      return;
    }

    if (this.manualClose) {
      this.setStatus('closed');
      return;
    }

    this.scheduleReconnect();
  }

  private scheduleReconnect() {
    this.setStatus('reconnecting');
    const delay = this.nextDelay();
    this.attempt += 1;
    log.info('WS переподключение', { attempt: this.attempt, delay });
    this.reconnectTimer = setTimeout(() => this.open(), delay);
  }

  private nextDelay() {
    const { baseDelay, maxDelay, factor, jitter } = WS_TUNING.reconnect;
    const raw = Math.min(maxDelay, baseDelay * factor ** this.attempt);
    const spread = raw * jitter * (Math.random() * 2 - 1);
    return Math.max(0, Math.round(raw + spread));
  }

  // Отсоединяем обработчики и закрываем сокет, чтобы его поздние события
  // (onclose/onmessage) уже не влияли на клиент.
  private teardownSocket() {
    const socket = this.socket;
    this.socket = null;
    if (!socket) return;
    socket.onopen = null;
    socket.onmessage = null;
    socket.onclose = null;
    socket.onerror = null;
    try {
      socket.close(NORMAL_CLOSURE, 'client closed');
    } catch (error) {
      log.debug('WS close при teardown не удался (не критично)', error);
    }
  }

  private enqueue(raw: string) {
    const { maxSize } = WS_TUNING.queue;
    this.queue.push({ raw, ts: Date.now() });
    // Вытесняем самые старые при переполнении.
    if (this.queue.length > maxSize) {
      this.queue.splice(0, this.queue.length - maxSize);
    }
  }

  private flushQueue() {
    if (this.socket?.readyState !== WebSocket.OPEN) return;
    const now = Date.now();
    const { ttl } = WS_TUNING.queue;
    const pending = this.queue;
    this.queue = [];
    for (const { raw, ts } of pending) {
      if (now - ts > ttl) continue; // протухло — не отправляем
      this.socket.send(raw);
    }
  }

  // Поддерживаем набор каналов, чтобы восстановить подписки после reconnect:
  // уже отправленные `subscribe` очередь не хранит.
  private trackSubscription(message: WsOutgoing) {
    if (message.type === 'subscribe') {
      this.channels.add(message.payload.channel);
    } else if (message.type === 'unsubscribe') {
      this.channels.delete(message.payload.channel);
    }
  }

  private resubscribe() {
    if (this.socket?.readyState !== WebSocket.OPEN) return;
    for (const channel of this.channels) {
      this.socket.send(
        JSON.stringify({ type: 'subscribe', payload: { channel } }),
      );
    }
  }

  private startHeartbeat() {
    this.stopHeartbeat();
    this.heartbeatTimer = setInterval(() => {
      this.send({ type: 'ping' });
      this.clearPongTimer();
      this.pongTimer = setTimeout(() => {
        log.warn('WS pong timeout — принудительный reconnect');
        this.socket?.close();
      }, WS_TUNING.heartbeat.pongTimeout);
    }, WS_TUNING.heartbeat.interval);
  }

  private stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
    this.clearPongTimer();
  }

  private clearPongTimer() {
    if (this.pongTimer) {
      clearTimeout(this.pongTimer);
      this.pongTimer = null;
    }
  }

  private clearTimers() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.stopHeartbeat();
  }

  private setStatus(status: WsStatus) {
    if (this.status === status) return;
    this.status = status;
    for (const listener of this.statusListeners) listener(status);
  }

  // Будим сокет сразу при возврате вкладки / восстановлении сети,
  // не дожидаясь текущего backoff-таймера.
  private onlineHandler = () => this.wake();
  private visibilityHandler = () => {
    if (document.visibilityState === 'visible') this.wake();
  };

  private bindWindowEvents() {
    window.addEventListener('online', this.onlineHandler);
    document.addEventListener('visibilitychange', this.visibilityHandler);
  }

  private unbindWindowEvents() {
    window.removeEventListener('online', this.onlineHandler);
    document.removeEventListener('visibilitychange', this.visibilityHandler);
  }

  private wake() {
    if (this.manualClose || this.isLive()) return;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.attempt = 0;
    this.open();
  }
}

/** Единый экземпляр на приложение. */
export const wsClient = new WsClient();
