import { createContext } from 'react';

import { type WsClient, wsClient } from './socket';

export const WsContext = createContext<WsClient>(wsClient);
