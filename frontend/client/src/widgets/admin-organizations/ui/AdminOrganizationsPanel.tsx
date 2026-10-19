import { useState } from 'react';
import { Link, useSearchParams } from 'react-router';

import {
  ReasonDialog,
  useApproveOrganization,
  useRejectOrganization,
} from '@/features/admin-moderation';

import {
  useAdminOrganization,
  useAdminOrganizations,
} from '@/entities/organization';

import { useOffsetPagination } from '@/shared/hooks';
import {
  Badge,
  Button,
  Input,
  Notice,
  Pagination,
  Skeleton,
} from '@/shared/ui';

const labels: Record<string, string> = {
  pending: 'Ждёт решения',
  approved: 'Допущена',
  rejected: 'Отклонена',
  published: 'Опубликовано',
  unpublished: 'Снято',
  cancelled: 'Отменено',
  deleted: 'Удалено',
};
export const AdminOrganizationsPanel = () => {
  const [params, setParams] = useSearchParams();
  const q = params.get('org_q') ?? '';
  const status = params.get('org_status') ?? 'pending';
  const pagination = useOffsetPagination(20, 'org_offset');
  const list = useAdminOrganizations({
    q,
    status: status === 'all' ? undefined : status,
    offset: pagination.offset,
    limit: 20,
  });
  const selectedId = params.get('org_id') ?? list.data?.items[0]?.id ?? '';
  const detail = useAdminOrganization(selectedId);
  const approve = useApproveOrganization();
  const reject = useRejectOrganization();
  const [rejectId, setRejectId] = useState('');
  const update = (key: string, value: string) =>
    setParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set(key, value);
      if (key !== 'org_id') {
        next.delete('org_offset');
        next.delete('org_id');
      }
      return next;
    });
  const org = detail.data;
  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-3xl font-bold">Допуск организаций</h2>
        <Badge variant="outline">Найдено: {list.data?.total ?? '…'}</Badge>
      </div>
      <div className="flex flex-wrap gap-3">
        <Input
          className="max-w-sm"
          aria-label="Поиск организаций"
          placeholder="Название, описание, город"
          value={q}
          onChange={(e) => update('org_q', e.target.value)}
        />
        <select
          className="rounded-xl border bg-card px-4"
          aria-label="Статус заявки"
          value={status}
          onChange={(e) => update('org_status', e.target.value)}
        >
          {[
            ['pending', 'Ждут решения'],
            ['all', 'Все'],
            ['approved', 'Допущены'],
            ['rejected', 'Отклонены'],
          ].map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </div>
      {list.isPending && <Skeleton className="h-40" />}
      {list.isError && (
        <Notice tone="danger">
          Не удалось загрузить заявки.{' '}
          <Button onClick={() => void list.refetch()}>Повторить</Button>
        </Notice>
      )}
      <div className="grid items-start gap-6 lg:grid-cols-[minmax(240px,1fr)_2fr]">
        <div className="space-y-3">
          {list.data?.items.map((item) => (
            <button
              key={item.id}
              onClick={() => update('org_id', item.id)}
              aria-pressed={selectedId === item.id}
              className={`w-full rounded-2xl border-2 bg-card p-5 text-left ${selectedId === item.id ? 'border-blue' : 'border-transparent'}`}
            >
              <span className="block text-xl font-bold">{item.name}</span>
              <span className="mt-2 block">
                {labels[item.status]} · {item.city}
              </span>
            </button>
          ))}
          {list.data?.total === 0 && (
            <p>Ничего не найдено. Измените поиск или фильтр.</p>
          )}
        </div>
        {selectedId && (
          <section
            className="space-y-6 rounded-3xl bg-card p-6 sm:p-8"
            aria-label="Карточка заявки"
          >
            {detail.isPending && <Skeleton className="h-60" />}
            {detail.isError && (
              <Notice tone="danger">
                Не удалось загрузить карточку.{' '}
                <Button onClick={() => void detail.refetch()}>Повторить</Button>
              </Notice>
            )}
            {org && (
              <>
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <h3 className="text-2xl font-bold">{org.name}</h3>
                  <Badge
                    variant={
                      (
                        {
                          approved: 'success',
                          pending: 'warning',
                          rejected: 'destructive',
                        } as const
                      )[org.status]
                    }
                  >
                    {labels[org.status]}
                  </Badge>
                </div>
                <dl className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <dt className="text-sm text-muted-foreground">
                      Представитель
                    </dt>
                    <dd>{org.owner?.name ?? 'Не указан'}</dd>
                  </div>
                  <div>
                    <dt className="text-sm text-muted-foreground">Контакты</dt>
                    <dd className="break-words whitespace-pre-wrap">
                      {org.contacts || 'Не указаны'}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-sm text-muted-foreground">
                      Направления
                    </dt>
                    <dd>
                      {org.sports.map((sport) => sport.name).join(', ') ||
                        'Не указаны'}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-sm text-muted-foreground">Город</dt>
                    <dd>{org.city || 'Не указан'}</dd>
                  </div>
                </dl>
                <div>
                  <h4 className="font-bold">Чем занимаются</h4>
                  <p className="whitespace-pre-wrap">{org.description}</p>
                </div>
                <div>
                  <h4 className="font-bold">Связанные встречи</h4>
                  {org.events.length ? (
                    <ul>
                      {org.events.map((event) => (
                        <li key={event.id}>
                          <Link
                            className="underline"
                            to={`/events/${event.id}`}
                          >
                            {event.title}
                          </Link>{' '}
                          · {labels[event.status] ?? event.status}
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p>Пока нет встреч.</p>
                  )}
                </div>
                <div>
                  <h4 className="font-bold">История допуска</h4>
                  <ul>
                    {org.history.map((entry, i) => (
                      <li className="border-b py-3" key={i}>
                        {new Date(entry.createdAt).toLocaleString('ru-RU')} ·{' '}
                        {labels[entry.from] ?? entry.from} →{' '}
                        {labels[entry.to] ?? entry.to}
                        <p>{entry.comment}</p>
                      </li>
                    ))}
                  </ul>
                </div>
                {org.status === 'pending' ? (
                  <div className="flex flex-wrap gap-3">
                    <Button
                      disabled={approve.isPending}
                      onClick={() => approve.mutate(org.id)}
                    >
                      Одобрить
                    </Button>
                    <Button
                      variant="destructive"
                      onClick={() => setRejectId(org.id)}
                    >
                      Отклонить с причиной
                    </Button>
                  </div>
                ) : (
                  <Notice
                    tone={org.status === 'approved' ? 'success' : 'neutral'}
                  >
                    {org.status === 'approved'
                      ? 'Организация допущена. Её встречи публикуются сразу.'
                      : `Заявка отклонена. ${org.reviewReason ?? ''}`}
                  </Notice>
                )}
              </>
            )}
          </section>
        )}
      </div>
      {list.data && (
        <Pagination
          label="Организации"
          total={list.data.total}
          hasPrev={pagination.hasPrev}
          hasNext={pagination.hasNext(list.data.total)}
          onPrev={pagination.toPrev}
          onNext={pagination.toNext}
        />
      )}
      <ReasonDialog
        key={rejectId}
        open={!!rejectId}
        onOpenChange={(open) => !open && setRejectId('')}
        title="Отклонить заявку"
        description="Причину увидит представитель; он сможет исправить профиль и подать заявку снова."
        submitLabel="Отклонить"
        pending={reject.isPending}
        error={reject.error}
        onSubmit={(reason) =>
          reject.mutate(
            { id: rejectId, reason },
            { onSuccess: () => setRejectId('') },
          )
        }
      />
    </div>
  );
};
