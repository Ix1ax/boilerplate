import { dayPartLabels, levelLabels, weekdayLabels } from '@starter/contracts';

import type { CompanyRequest } from '@/entities/company';

import { Badge, Button } from '@/shared/ui';

import { Matches } from './Matches';

interface CompanyRequestCardProps {
  item: CompanyRequest;
  selected: boolean;
  closing: boolean;
  onToggle: () => void;
  onClose: () => void;
}

export const CompanyRequestCard = ({
  item,
  selected,
  closing,
  onToggle,
  onClose,
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
        {item.area && ` · ${item.area}`}
      </p>
      <p>
        {item.weekdays.length
          ? item.weekdays.map((day) => weekdayLabels[day - 1]).join(', ')
          : 'Любой день'}{' '}
        ·{' '}
        {item.day_parts.length
          ? item.day_parts.map((part) => dayPartLabels[part]).join(', ')
          : 'Любое время'}
      </p>
      {item.note && <p>{item.note}</p>}
      <p>
        Подходящих встреч: {item.matches_count} · Ещё ищут компанию:{' '}
        {item.others_count}
      </p>
      {item.status === 'active' && (
        <p className="text-sm text-muted-foreground">
          Запрос действует до{' '}
          {new Date(item.expires_at).toLocaleDateString('ru-RU')}
        </p>
      )}
      <div className="flex flex-wrap gap-3">
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
      {selected && <Matches id={item.id} />}
    </article>
  );
};
