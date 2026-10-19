import { Link } from 'react-router';
import { useQuery } from '@tanstack/react-query';

import { getCompanyDemand } from '@/entities/company';
import type { Sport } from '@/entities/sport';

import { STATIC_ROUTES } from '@/shared/constants';
import { Badge, Button, Eyebrow, Notice } from '@/shared/ui';
const when = (s: string) =>
  new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'Asia/Krasnoyarsk',
    day: 'numeric',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(s));
export const ClubDemandPanel = ({ sports }: { sports: Sport[] }) => {
  const q = useQuery({
    queryKey: ['company', 'club-demand'],
    queryFn: getCompanyDemand,
    meta: { silent: true },
  });
  const items =
    q.data?.items.filter(
      (p) => !sports.length || sports.some((s) => s.id === p.sportId),
    ) ?? [];
  return (
    <section className="grid gap-4">
      <div>
        <Eyebrow className="text-blue">для клубов</Eyebrow>
        <h2 className="title-display mt-2 text-3xl">Спрос рядом</h2>
        <p>
          Сколько людей ищут спорт в ваших зонах и когда им удобно. Без имён,
          контактов и личных расписаний.
        </p>
      </div>
      {q.isPending && <p role="status">Загружаем спрос…</p>}
      {q.isError && (
        <Notice tone="warning">
          Спрос доступен после допуска организации.{' '}
          <Button onClick={() => void q.refetch()}>Повторить</Button>
        </Notice>
      )}
      {!q.isPending && !q.isError && !items.length && (
        <p>Пока нет групп из трёх запросов с общим часом.</p>
      )}
      <p className="text-sm text-muted-foreground">
        Группы меньше 3 запросов не раскрываем: нельзя узнать число и окна
        конкретного человека.
      </p>
      <div className="grid gap-4 md:grid-cols-2">
        {items.map((p) => {
          const params = new URLSearchParams({
            sport_id: p.sportId,
            sport_name: p.sportName,
            zone: p.zone,
            starts_at: p.startsAt,
            ends_at: p.endsAt,
            level: p.level,
            as_organization: '1',
          });
          return (
            <article
              key={p.version}
              className="grid gap-3 rounded-3xl bg-card p-6"
            >
              <Badge>{p.count} активных запросов</Badge>
              <h3 className="text-xl font-bold">
                {p.sportName} · {p.zone}
              </h3>
              <b>
                Общий час: {when(p.startsAt)} — {when(p.endsAt)}
              </b>
              <p>
                У каждого окно целиком покрывает общий час. Отозванные и
                истёкшие запросы не учтены.
              </p>
              <p>
                Публикация никого не записывает. Решение публиковать остаётся за
                вами.
              </p>
              <Button
                render={<Link to={`${STATIC_ROUTES.EVENT_NEW}?${params}`} />}
              >
                Создать встречу под этот спрос
              </Button>
            </article>
          );
        })}
      </div>
    </section>
  );
};
