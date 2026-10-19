import { Link } from 'react-router';
import { joinModeLabels, levelLabels } from '@starter/contracts';

import { useAdminComments } from '@/entities/admin';
import { useEvent } from '@/entities/event';

import { Button, Notice, Skeleton } from '@/shared/ui';

interface AdminEventDetailProps {
  id: string;
  onUnpublish: (id: string) => void;
  onDelete: (id: string) => void;
  onRestore: (id: string) => void;
  onHistory: (id: string) => void;
  pending: boolean;
}
export const AdminEventDetail = ({
  id,
  onUnpublish,
  onDelete,
  onRestore,
  onHistory,
  pending,
}: AdminEventDetailProps) => {
  const query = useEvent(id);
  const comments = useAdminComments({ event_id: id });
  if (query.isPending) return <Skeleton className="h-80" />;
  if (query.isError)
    return (
      <Notice tone="danger">
        Не удалось загрузить встречу.{' '}
        <Button onClick={() => void query.refetch()}>Повторить</Button>
      </Notice>
    );
  const event = query.data;
  return (
    <section
      className="space-y-6 rounded-3xl bg-card p-6 sm:p-8"
      aria-label="Карточка мероприятия"
    >
      <h3 className="text-3xl font-bold">{event.title}</h3>
      <dl className="grid gap-5 sm:grid-cols-2">
        <div>
          <dt className="text-sm text-muted-foreground">Организатор</dt>
          <dd>
            {event.organizer.name} ·{' '}
            {event.organizer.type === 'organization' ? 'Клуб' : 'Участник'}
          </dd>
        </div>
        <div>
          <dt className="text-sm text-muted-foreground">Когда и где</dt>
          <dd>
            {new Date(event.startsAt).toLocaleString('ru-RU')}
            <p>{event.location}</p>
          </dd>
        </div>
        <div>
          <dt className="text-sm text-muted-foreground">Участие</dt>
          <dd>
            {joinModeLabels[event.joinMode]} · {levelLabels[event.level]}
          </dd>
        </div>
        <div>
          <dt className="text-sm text-muted-foreground">Направление</dt>
          <dd>{event.sport.name}</dd>
        </div>
      </dl>
      <p className="whitespace-pre-wrap">{event.description}</p>
      <div>
        <h4 className="font-bold">Обсуждение · последнее</h4>
        {comments.isPending && <p>Загружаем…</p>}
        {!comments.isPending && comments.isError && (
          <Button onClick={() => void comments.refetch()}>
            Повторить загрузку
          </Button>
        )}
        {!comments.isPending && !comments.isError && (
          <p className="whitespace-pre-wrap">
            {comments.data?.items[0]?.body ?? 'Сообщений пока нет.'}
          </p>
        )}
      </div>
      <Button variant="outline" render={<Link to={`/events/${id}`} />}>
        Открыть встречу и обсуждение
      </Button>
      {event.moderationReason && (
        <Notice tone="warning">
          Причина модерации: {event.moderationReason}
        </Notice>
      )}
      <div className="flex flex-wrap gap-3">
        {event.status === 'published' && (
          <Button onClick={() => onUnpublish(id)}>Снять с публикации</Button>
        )}
        {event.status === 'unpublished' && (
          <Button disabled={pending} onClick={() => onRestore(id)}>
            Вернуть публикацию
          </Button>
        )}
        {event.status !== 'deleted' && (
          <Button variant="destructive" onClick={() => onDelete(id)}>
            Удалить
          </Button>
        )}
        <Button variant="ghost" onClick={() => onHistory(id)}>
          История
        </Button>
      </div>
      <p className="text-sm text-muted-foreground">
        Снятие убирает встречу из выдачи и закрывает новые записи. Организатор и
        участники получат причину.
      </p>
    </section>
  );
};
