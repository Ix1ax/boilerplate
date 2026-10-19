import { useForm } from 'react-hook-form';
import { Link } from 'react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  companyResponseSchema,
  type CompanyResponseValues,
  levelLabels,
} from '@starter/contracts';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  COMPANY_KEYS,
  getCompanyProposal,
  respondCompanyProposal,
} from '@/entities/company';

import { DYNAMIC_ROUTES, STATIC_ROUTES } from '@/shared/constants';
import { Button, Notice } from '@/shared/ui';
const when = (s: string) =>
  new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'Asia/Krasnoyarsk',
    day: 'numeric',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(s));
export const CommonProposal = ({
  id,
  onEdit,
}: {
  id: string;
  onEdit: () => void;
}) => {
  const answer = useForm<CompanyResponseValues>({
    resolver: zodResolver(companyResponseSchema),
  });
  const send = (version: string, interested: boolean) => {
    answer.setValue('version', version);
    answer.setValue('interested', interested);
    void answer.handleSubmit((values) => response.mutate(values))();
  };
  const cache = useQueryClient();
  const key = [...COMPANY_KEYS.ALL, 'proposal', id];
  const query = useQuery({
    queryKey: key,
    queryFn: () => getCompanyProposal(id),
    meta: { silent: true },
  });
  const response = useMutation({
    mutationFn: ({
      version,
      interested,
    }: {
      version: string;
      interested: boolean;
    }) => respondCompanyProposal(id, version, interested),
    onSuccess: () => cache.invalidateQueries({ queryKey: key }),
    onError: () => {
      void cache.invalidateQueries({ queryKey: key });
    },
    meta: { silent: true },
  });
  if (query.isPending) return <p role="status">Ищем общий вариант…</p>;
  if (query.isError)
    return (
      <Notice tone="danger">
        Не удалось загрузить общий вариант.{' '}
        <Button onClick={() => void query.refetch()}>Повторить</Button>
      </Notice>
    );
  const p = query.data?.proposal;
  if (!p)
    return (
      <section className="grid gap-3 rounded-2xl bg-background p-5">
        <h3 className="text-xl font-bold">Пока нет общего варианта</h3>
        <p>
          Мы не придумываем участников: вариант появится, когда совпадут
          реальные запросы.
        </p>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={onEdit}>
            Добавить окно или соседнюю зону
          </Button>
          <Button
            variant="outline"
            render={<Link to={STATIC_ROUTES.EVENT_NEW} />}
          >
            Создать встречу самому
          </Button>
        </div>
      </section>
    );
  return (
    <section className="grid gap-4 rounded-2xl bg-background p-5">
      <span className="font-mono text-sm text-muted-foreground">
        Общий вариант · версия {p.version.slice(0, 8)}
      </span>
      <h3 className="text-2xl font-bold">
        {p.sportName} · {when(p.startsAt)}
      </h3>
      <p>
        Зона {p.zone} · {levelLabels[p.level]} · до {when(p.endsAt)}
      </p>
      <div className="flex flex-wrap gap-4">
        <b>Заинтересовались: {p.interested}</b>
        <b>Записались: {p.enrolled}</b>
      </div>
      <p>
        Эти числа нельзя складывать: интерес — это не запись. Имена и контакты
        заинтересованных не раскрываем.
      </p>
      <h4 className="font-bold">Почему этот вариант</h4>
      <ul className="grid gap-2">
        <li>✓ Все выбрали {p.sportName}</li>
        <li>✓ Окно каждого целиком покрывает этот час</li>
        <li>✓ Все согласились на зону {p.zone} и совместимый уровень</li>
      </ul>
      <p className="text-sm text-muted-foreground">
        Подбор работает по правилам, без процентов совместимости и без ИИ. После
        изменения условий нужен новый ответ.
      </p>
      {p.eventId ? (
        <Notice tone="success">
          <b>Встреча создана</b>
          <p>Интерес не записывает вас в состав. Запишитесь отдельно.</p>
          <Button render={<Link to={DYNAMIC_ROUTES.EVENT(p.eventId)} />}>
            Открыть и записаться
          </Button>
        </Notice>
      ) : (
        <>
          <h4 className="font-bold">Вам подходит?</h4>
          {p.response !== null && (
            <p role="status">
              {p.response
                ? '◎ Вы заинтересованы. Это ещё не запись'
                : 'Вы ответили «Не подходит»'}
            </p>
          )}
          <div className="flex flex-wrap gap-2">
            <Button
              disabled={response.isPending}
              onClick={() => send(p.version, true)}
            >
              Мне подходит
            </Button>
            <Button
              variant="outline"
              disabled={response.isPending}
              onClick={() => send(p.version, false)}
            >
              Не подходит
            </Button>
          </div>
        </>
      )}
      {response.isError && (
        <Notice tone="danger" role="alert">
          {response.error.message} Условия могли измениться — проверьте
          актуальный вариант.
        </Notice>
      )}
    </section>
  );
};
