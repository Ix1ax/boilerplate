import { type ReactNode, useEffect } from 'react';

import { getWsConfig } from './config';
import { WsContext } from './context';
import { type WsClient, wsClient } from './socket';

interface WsProviderProps {
  client?: WsClient;
  children?: ReactNode;
}

/**
 * Держит единый сокет. Подключается на mount только при наличии токена
 * (на логине без сессии анонимный сокет не открываем). Смена сессии —
 * `client.reconnect()`/`client.close()` из features/auth. Закрытие на unmount.
 */
export const WsProvider = ({
  client = wsClient,
  children,
}: WsProviderProps) => {
  useEffect(() => {
    if (getWsConfig().getToken()) client.connect();
    return () => client.close();
  }, [client]);

  return <WsContext value={client}>{children}</WsContext>;
};
