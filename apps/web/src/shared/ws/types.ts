import { z } from 'zod';

// Протокол сообщений. Расширяй zod-схемы под свой бэкенд — типы (`WsIncoming`/
// `WsOutgoing`) выводятся из них, а runtime-валидатор (`parseIncoming` в
// socket.ts) остаётся синхронным с типами автоматически.

export type WsStatus =
  'idle' | 'connecting' | 'open' | 'reconnecting' | 'closed';

/** Входящие сообщения (сервер → клиент). */
export const wsIncomingSchema = z.discriminatedUnion('type', [
  z.object({ type: z.literal('pong') }),
  z.object({
    type: z.literal('notification'),
    payload: z.object({ id: z.string(), message: z.string() }),
  }),
]);

/** Исходящие сообщения (клиент → сервер). */
export const wsOutgoingSchema = z.discriminatedUnion('type', [
  z.object({ type: z.literal('ping') }),
  z.object({
    type: z.literal('subscribe'),
    payload: z.object({ channel: z.string() }),
  }),
  z.object({
    type: z.literal('unsubscribe'),
    payload: z.object({ channel: z.string() }),
  }),
]);

export type WsIncoming = z.infer<typeof wsIncomingSchema>;
export type WsOutgoing = z.infer<typeof wsOutgoingSchema>;

/** Конкретное входящее сообщение по его type. */
export type WsMessageOf<T extends WsIncoming['type']> = Extract<
  WsIncoming,
  { type: T }
>;

/** Обработчик входящего сообщения конкретного типа. */
export type WsHandler<T extends WsIncoming['type']> = (
  message: WsMessageOf<T>,
) => void;
