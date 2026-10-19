import { useState } from 'react';

import { useCompanyMatches } from '@/entities/company';
import { EventCard } from '@/entities/event';

import { Button, Notice } from '@/shared/ui';

interface MatchesProps {
  id: string;
}

export const Matches = ({ id }: MatchesProps) => {
  const [offset, setOffset] = useState(0);
  const matches = useCompanyMatches(id, offset);
  if (matches.isPending) return <p role="status">Ищем подходящие встречи…</p>;
  if (matches.isError)
    return (
      <Notice tone="danger" role="alert">
        Не удалось загрузить встречи.{' '}
        <Button onClick={() => void matches.refetch()}>Повторить</Button>
      </Notice>
    );
  return (
    <div className="flex flex-col gap-4">
      {!matches.data.items.length && (
        <p>
          Подходящих встреч пока нет. Когда появятся новые, мы пришлём
          уведомление.
        </p>
      )}
      <div className="grid gap-4 md:grid-cols-2">
        {matches.data.items.map((event) => (
          <EventCard key={event.id} event={event} />
        ))}
      </div>
      <div className="flex gap-3">
        {offset > 0 && (
          <Button onClick={() => setOffset(offset - 12)}>Назад</Button>
        )}
        {offset + 12 < matches.data.total && (
          <Button onClick={() => setOffset(offset + 12)}>Далее</Button>
        )}
      </div>
    </div>
  );
};
