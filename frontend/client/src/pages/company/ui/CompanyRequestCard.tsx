import { dayPartLabels, levelLabels, weekdayLabels } from '@starter/contracts';

import type { CompanyRequest } from '@/entities/company';

import { Badge, Button } from '@/shared/ui';

import { CommonProposal } from './CommonProposal';
import { Matches } from './Matches';

interface CompanyRequestCardProps {
  item: CompanyRequest;
  selected: boolean;
  closing: boolean;
  onToggle: () => void;
  onClose: () => void;
  onEdit: () => void;
}

export const CompanyRequestCard = ({
  item,
  selected,
  closing,
  onToggle,
  onClose,
  onEdit,
}: CompanyRequestCardProps) => {
  return (
    <article className="flex flex-col gap-4 rounded-3xl bg-card p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-xl font-bold">{item.sport.name}</h3>
        <Badge variant={item.status === 'active' ? 'interest' : 'neutral'}>
          {
            {
              active: 'Ищем компанию',
              closed: 'Закрыт',
              expired: 'Срок истёк',
            }[item.status]
          }
        </Badge>
      </div>
      <p>
        {levelLabels[item.level]}
        {item.zones?.length
          ? ` · ${item.zones.join(', ')}`
          : item.area && ` · ${item.area}`}
      </p>
      {item.windows?.length > 0 && (
        <ul>
          {item.windows.map((w, i) => (
            <li key={i}>
              {new Date(w.startsAt).toLocaleString('ru-RU', {
                timeZone: 'Asia/Krasnoyarsk',
              })}{' '}
              —{' '}
              {new Date(w.endsAt).toLocaleTimeString('ru-RU', {
                timeZone: 'Asia/Krasnoyarsk',
                hour: '2-digit',
                minute: '2-digit',
              })}
            </li>
          ))}
        </ul>
      )}
      {!item.windows?.length && (
        <p>
          {item.weekdays.length
            ? item.weekdays.map((day) => weekdayLabels[day - 1]).join(', ')
            : 'Любой день'}{' '}
          ·{' '}
          {item.dayParts.length
            ? item.dayParts.map((part) => dayPartLabels[part]).join(', ')
            : 'Любое время'}
        </p>
      )}
      {item.note && <p>{item.note}</p>}
      <p>
        Подходящих встреч: {item.matchesCount} · Ещё ищут компанию:{' '}
        {item.othersCount}
      </p>
      {item.status === 'active' && (
        <p className="text-sm text-muted-foreground">
          Запрос действует до{' '}
          {new Date(item.expiresAt).toLocaleDateString('ru-RU')}
        </p>
      )}
      <div className="flex flex-wrap gap-3">
        {item.status === 'active' && (
          <Button variant="outline" onClick={onEdit}>
            Изменить
          </Button>
        )}
        <Button onClick={() => onToggle()}>
          {selected ? 'Скрыть встречи' : 'Посмотреть встречи'}
        </Button>
        {item.status === 'active' && (
          <Button
            variant="outline"
            disabled={closing}
            onClick={() => onClose()}
          >
            Больше не ищу
          </Button>
        )}
      </div>
      {selected && (
        <>
          <CommonProposal id={item.id} onEdit={onEdit} />
          <Matches id={item.id} />
        </>
      )}
    </article>
  );
};
