import {
  use,
  useCallback,
  useEffect,
  useRef,
  useSyncExternalStore,
} from 'react';
import { type QueryKey, useQueryClient } from '@tanstack/react-query';

import { WsContext } from './context';
import type { WsHandler, WsIncoming, WsOutgoing } from './types';

export const useWsClient = () => use(WsContext);

/** Текущий статус соединения (реактивно). */
export const useWsStatus = () => {
  const client = useWsClient();
  return useSyncExternalStore(
    (onChange) => client.onStatus(onChange),
    () => client.getStatus(),
  );
};

/** Стабильная функция отправки. */
export const useWsSend = () => {
  const client = useWsClient();
  return useCallback((message: WsOutgoing) => client.send(message), [client]);
};

/** Подписка на тип события с авто-отпиской. Handler можно передавать инлайном. */
export const useWsSubscription = <T extends WsIncoming['type']>(
  type: T,
  handler: WsHandler<T>,
) => {
  const client = useWsClient();
  const ref = useRef(handler);

  useEffect(() => {
    ref.current = handler;
  });

  useEffect(
    () => client.on(type, (message) => ref.current(message)),
    [client, type],
  );
};

/** Сахар: инвалидировать queryKey по приходу WS-события. */
export const useWsInvalidate = <T extends WsIncoming['type']>(
  type: T,
  queryKey: QueryKey,
) => {
  const queryClient = useQueryClient();
  useWsSubscription(type, () => {
    void queryClient.invalidateQueries({ queryKey });
  });
};
