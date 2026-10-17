import type { QueryClient } from '@tanstack/react-query';

import type { WsClient } from './socket';
import type { WsIncoming, WsMessageOf } from './types';

/** Таблица: тип WS-события → что сделать с кэшем TanStack Query. */
export type WsQueryBinding = {
  [T in WsIncoming['type']]?: (
    message: WsMessageOf<T>,
    queryClient: QueryClient,
  ) => void;
};

/**
 * Декларативно связывает WS-события с кэшем Query.
 * Возвращает функцию для снятия всех подписок.
 *
 * @example
 * bindWsToQuery(wsClient, queryClient, {
 *   notification: (_msg, qc) => qc.invalidateQueries({ queryKey: ['notifications'] }),
 * })
 */
export const bindWsToQuery = (
  client: WsClient,
  queryClient: QueryClient,
  bindings: WsQueryBinding,
) => {
  const types = Object.keys(bindings) as WsIncoming['type'][];
  const unsubs = types.map((type) => {
    const handler = bindings[type]!;
    return client.on(type, (message) => handler(message as never, queryClient));
  });
  return () => unsubs.forEach((unsub) => unsub());
};
