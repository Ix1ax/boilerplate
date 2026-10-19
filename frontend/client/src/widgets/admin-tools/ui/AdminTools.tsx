import { useState } from 'react';
import { Link, useSearchParams } from 'react-router';
import { accountBlockSchema } from '@starter/contracts';

import { ReasonDialog } from '@/features/admin-moderation';

import {
  useAdminAccount,
  useAdminAccounts,
  useAdminAction,
  useAdminComments,
  useAdminLog,
  useAdminSports,
} from '@/entities/admin';

import { useOffsetPagination } from '@/shared/hooks';
import { Button, Input, Notice, Pagination, Skeleton } from '@/shared/ui';

const stateLabels: Record<string, string> = {
  pending: 'На рассмотрении',
  approved: 'Допущена',
  rejected: 'Отклонена',
  published: 'Опубликовано',
  unpublished: 'Снято',
  cancelled: 'Отменено',
  deleted: 'Удалено',
  visible: 'Видимо',
  hidden: 'Скрыто',
  gathering: 'Собираем состав',
  reached: 'Состав собран',
  failed: 'Состав не собран',
  registered: 'Регистрация',
  blocked: 'Блокировка',
  unblocked: 'Разблокировка',
  password_changed: 'Пароль изменён',
  none: '—',
};

const useResetOffset = (key: string) => {
  const [, setParams] = useSearchParams();
  return () =>
    setParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete(key);
      return next;
    });
};

export const AdminAccountsPanel = () => {
  const resetOffset = useResetOffset('accounts_offset');
  const [q, setQ] = useState('');
  const [status, setStatus] = useState('');
  const [id, setId] = useState('');
  const [blockId, setBlockId] = useState('');
  const pagination = useOffsetPagination(20, 'accounts_offset');
  const list = useAdminAccounts({ q, status, offset: pagination.offset });
  const detail = useAdminAccount(id);
  const action = useAdminAction();
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-3">
        <Input
          aria-label="Поиск пользователей"
          placeholder="Почта или имя"
          maxLength={100}
          value={q}
          onChange={(e) => {
            setQ(e.target.value);
            resetOffset();
          }}
        />
        <select
          aria-label="Статус пользователя"
          value={status}
          onChange={(e) => {
            setStatus(e.target.value);
            resetOffset();
          }}
        >
          <option value="">Все</option>
          <option value="active">Активные</option>
          <option value="blocked">Заблокированные</option>
        </select>
      </div>
      <p className="text-sm text-muted-foreground">
        Учётные записи обычной регистрации. Доступ сотрудников через Столото ID
        управляется отдельно.
      </p>
      {list.isPending && <Skeleton className="h-40" />}
      {list.isError && (
        <Notice tone="danger">
          Не удалось загрузить пользователей.{' '}
          <Button onClick={() => void list.refetch()}>Повторить</Button>
        </Notice>
      )}
      <div className="grid gap-5 lg:grid-cols-2">
        <div className="space-y-3">
          {list.data?.items.map((account) => (
            <article className="rounded-2xl border p-4" key={account.id}>
              <button
                className="font-bold underline"
                onClick={() => setId(account.id)}
              >
                {account.name}
              </button>
              <p>{account.email}</p>
              <p>
                {account.status === 'blocked'
                  ? `Заблокирован: ${account.blockReason}`
                  : 'Активен'}
              </p>
              <Button
                disabled={action.isPending}
                variant="outline"
                onClick={() =>
                  account.status === 'blocked'
                    ? action.mutate({
                        url: `/admin/accounts/${account.id}/unblock`,
                      })
                    : setBlockId(account.id)
                }
              >
                {account.status === 'blocked'
                  ? 'Разблокировать'
                  : 'Заблокировать'}
              </Button>
            </article>
          ))}
          {list.data?.total === 0 && <p>Пользователи не найдены.</p>}
        </div>
        {id && (
          <section className="rounded-2xl bg-card p-5">
            <h3 className="text-xl font-bold">История учётной записи</h3>
            {detail.isPending && <p>Загружаем…</p>}
            {detail.isError && (
              <Button onClick={() => void detail.refetch()}>
                Повторить загрузку
              </Button>
            )}
            {detail.data && (
              <>
                <p>
                  {detail.data.name} · {detail.data.email}
                </p>
                <ul>
                  {detail.data.history.map((entry, i) => (
                    <li className="border-b py-3" key={i}>
                      {new Date(entry.createdAt).toLocaleString('ru-RU')} ·{' '}
                      {stateLabels[entry.action] ?? entry.action}
                      <p>{entry.comment}</p>
                    </li>
                  ))}
                </ul>
              </>
            )}
          </section>
        )}
      </div>
      {list.data && (
        <Pagination
          label="Пользователи"
          total={list.data.total}
          hasPrev={pagination.hasPrev}
          hasNext={pagination.hasNext(list.data.total)}
          onPrev={pagination.toPrev}
          onNext={pagination.toNext}
        />
      )}
      <ReasonDialog
        schema={accountBlockSchema}
        key={blockId}
        open={!!blockId}
        onOpenChange={(open) => !open && setBlockId('')}
        title="Заблокировать пользователя"
        description="Пользователь увидит причину при входе."
        submitLabel="Заблокировать"
        pending={action.isPending}
        error={action.error}
        destructive
        onSubmit={(reason) =>
          action.mutate(
            { url: `/admin/accounts/${blockId}/block`, reason },
            { onSuccess: () => setBlockId('') },
          )
        }
      />
    </div>
  );
};

export const AdminSportsPanel = () => {
  const list = useAdminSports();
  const action = useAdminAction();
  const [id, setId] = useState('');
  return (
    <div className="space-y-3">
      <h2 className="text-2xl font-bold">Справочник направлений</h2>
      {list.isPending && <Skeleton className="h-40" />}
      {list.isError && (
        <Button onClick={() => void list.refetch()}>Повторить загрузку</Button>
      )}
      {list.data?.items.map((sport) => (
        <article
          className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border p-4"
          key={sport.id}
        >
          <div>
            <b>{sport.name}</b>
            <p>
              Встреч: {sport.eventsCount} · запросов: {sport.requestsCount} ·{' '}
              {sport.hidden ? 'Скрыто' : 'В справочнике'}
            </p>
          </div>
          {sport.isCustom ? (
            <Button
              variant="outline"
              disabled={action.isPending}
              onClick={() =>
                sport.hidden
                  ? action.mutate({ url: `/admin/sports/${sport.id}/show` })
                  : setId(sport.id)
              }
            >
              {sport.hidden ? 'Вернуть' : 'Скрыть'}
            </Button>
          ) : (
            <span>Базовое направление</span>
          )}
        </article>
      ))}
      <ReasonDialog
        key={id}
        open={!!id}
        onOpenChange={(open) => !open && setId('')}
        title="Скрыть направление"
        submitLabel="Скрыть"
        pending={action.isPending}
        error={action.error}
        onSubmit={(reason) =>
          action.mutate(
            { url: `/admin/sports/${id}/hide`, reason },
            { onSuccess: () => setId('') },
          )
        }
      />
    </div>
  );
};

export const AdminCommentsPanel = () => {
  const resetOffset = useResetOffset('comments_offset');
  const [q, setQ] = useState('');
  const [status, setStatus] = useState('');
  const [id, setId] = useState('');
  const pagination = useOffsetPagination(20, 'comments_offset');
  const list = useAdminComments({ q, status, offset: pagination.offset });
  const action = useAdminAction();
  return (
    <div className="space-y-4">
      <div className="flex gap-3">
        <Input
          aria-label="Поиск сообщений"
          placeholder="Поиск сообщений"
          value={q}
          onChange={(e) => {
            setQ(e.target.value);
            resetOffset();
          }}
        />
        <select
          aria-label="Статус сообщения"
          value={status}
          onChange={(e) => {
            setStatus(e.target.value);
            resetOffset();
          }}
        >
          <option value="">Все</option>
          <option value="visible">Видимые</option>
          <option value="hidden">Скрытые</option>
          <option value="deleted">Удалённые</option>
        </select>
      </div>
      {list.isPending && <Skeleton className="h-40" />}
      {list.isError && (
        <Button onClick={() => void list.refetch()}>Повторить загрузку</Button>
      )}
      {list.data?.total === 0 && <p>Сообщений не найдено.</p>}
      {list.data?.items.map((comment) => (
        <article className="space-y-2 rounded-2xl border p-5" key={comment.id}>
          <Link
            className="font-bold underline"
            to={`/events/${comment.eventId}`}
          >
            {comment.eventTitle}
          </Link>
          <p className="text-sm">
            {comment.authorName} ·{' '}
            {new Date(comment.createdAt).toLocaleString('ru-RU')}
          </p>
          <p className="whitespace-pre-wrap break-words">{comment.body}</p>
          {comment.hiddenReason && (
            <p>Причина скрытия: {comment.hiddenReason}</p>
          )}
          {comment.status !== 'deleted' && (
            <Button
              variant="outline"
              disabled={action.isPending}
              onClick={() =>
                comment.status === 'hidden'
                  ? action.mutate({
                      url: `/admin/comments/${comment.id}/restore`,
                    })
                  : setId(comment.id)
              }
            >
              {comment.status === 'hidden'
                ? 'Вернуть сообщение'
                : 'Скрыть с причиной'}
            </Button>
          )}
        </article>
      ))}
      {list.data && (
        <Pagination
          label="Сообщения"
          total={list.data.total}
          hasPrev={pagination.hasPrev}
          hasNext={pagination.hasNext(list.data.total)}
          onPrev={pagination.toPrev}
          onNext={pagination.toNext}
        />
      )}
      <ReasonDialog
        key={id}
        open={!!id}
        onOpenChange={(open) => !open && setId('')}
        title="Скрыть сообщение"
        description="Остальное обсуждение сохранится. Открытые жалобы будут решены."
        submitLabel="Скрыть"
        pending={action.isPending}
        error={action.error}
        onSubmit={(reason) =>
          action.mutate(
            { url: `/admin/comments/${id}/hide`, reason },
            { onSuccess: () => setId('') },
          )
        }
      />
    </div>
  );
};

export const AdminModerationLog = () => {
  const resetOffset = useResetOffset('log_offset');
  const [q, setQ] = useState('');
  const pagination = useOffsetPagination(20, 'log_offset');
  const list = useAdminLog({ q, offset: pagination.offset });
  return (
    <div className="space-y-4">
      <Input
        aria-label="Поиск решений"
        placeholder="Поиск решений"
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          resetOffset();
        }}
      />
      {list.isPending && <Skeleton className="h-40" />}
      {list.isError && (
        <Button onClick={() => void list.refetch()}>Повторить загрузку</Button>
      )}
      {list.data?.total === 0 && <p>Решений не найдено.</p>}
      {list.data?.items.map((entry, index) => (
        <article
          className="rounded-2xl border p-4"
          key={`${entry.entityId}-${index}`}
        >
          <b>{entry.title}</b>
          <p>
            {stateLabels[entry.from] ?? entry.from} →{' '}
            {stateLabels[entry.to] ?? entry.to}
          </p>
          <p>{entry.comment}</p>
          <p className="text-sm text-muted-foreground">
            {entry.actor} · {new Date(entry.at).toLocaleString('ru-RU')}
          </p>
        </article>
      ))}
      {list.data && (
        <Pagination
          label="Решения"
          total={list.data.total}
          hasPrev={pagination.hasPrev}
          hasNext={pagination.hasNext(list.data.total)}
          onPrev={pagination.toPrev}
          onNext={pagination.toNext}
        />
      )}
    </div>
  );
};
