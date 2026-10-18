import { type FormEvent, useState } from 'react';

import { useCurrentUser } from '@/entities/user';

import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Input,
} from '@/shared/ui';
import { useChannel, useWsStatus } from '@/shared/ws';

type ChatMessage = { text: string; name: string };

const STATUS_LABEL = {
  idle: 'не подключено',
  connecting: 'подключение…',
  open: 'онлайн',
  reconnecting: 'переподключение…',
  closed: 'отключено',
} as const;

/** Образец realtime: общая комната room:demo. Откройте страницу в двух вкладках или под двумя пользователями. */
export const DashboardLivePage = () => {
  const status = useWsStatus();
  const { data: user } = useCurrentUser();
  const { messages, publish } = useChannel<ChatMessage>('room:demo');
  const [text, setText] = useState('');

  const send = (event: FormEvent) => {
    event.preventDefault();
    const value = text.trim();
    if (!value) return;
    publish({ text: value.slice(0, 500), name: user?.name ?? 'Аноним' });
    setText('');
  };

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between">
        <CardTitle>Комната room:demo</CardTitle>
        <Badge variant={status === 'open' ? 'secondary' : 'outline'}>
          {STATUS_LABEL[status]}
        </Badge>
      </CardHeader>
      <CardContent className="space-y-3">
        <ul
          aria-live="polite"
          className="max-h-80 space-y-2 overflow-y-auto text-sm"
        >
          {messages.length === 0 && (
            <li className="text-muted-foreground">
              Сообщений пока нет — откройте страницу во второй вкладке и
              напишите.
            </li>
          )}
          {messages.map((message, index) => (
            <li key={`${message.at}-${index}`}>
              <span className="font-medium">{message.data.name}:</span>{' '}
              {message.data.text}
            </li>
          ))}
        </ul>
        <form onSubmit={send} className="flex gap-2">
          <Input
            value={text}
            onChange={(event) => setText(event.target.value)}
            placeholder="Сообщение"
            aria-label="Сообщение в комнату"
            maxLength={500}
          />
          <Button type="submit" disabled={status !== 'open'}>
            Отправить
          </Button>
        </form>
      </CardContent>
    </Card>
  );
};
