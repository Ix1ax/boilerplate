import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Link } from 'react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { commentSchema, type CommentValues } from '@starter/contracts';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { cn } from 'cn';
import { motion } from 'motion/react';

import { useConfirm } from '@/features/confirm';
import { ReportButton } from '@/features/report';

import { CITY_TIME_ZONE, toCityDate } from '@/entities/event';
import { useCurrentUser } from '@/entities/user';

import { DYNAMIC_ROUTES, STATIC_ROUTES } from '@/shared/constants';
import { initialOf } from '@/shared/lib';
import { TRANSITION } from '@/shared/motion';
import { Badge, Button, Label, Skeleton, Textarea } from '@/shared/ui';

import { deleteComment, type EventComment } from '../api/commentApi';
import { useAddComment, useComments } from '../model';
import { COMMENT_KEYS } from '../model/useComments';

export const DISCUSSION_INPUT_ID = 'discussion-input';

const timeFmt = new Intl.DateTimeFormat('ru-RU', {
  timeZone: CITY_TIME_ZONE,
  hour: '2-digit',
  minute: '2-digit',
});
const dayFmt = new Intl.DateTimeFormat('ru-RU', {
  timeZone: CITY_TIME_ZONE,
  day: 'numeric',
  month: 'long',
});

const formatMeta = (value: string) => {
  const day = toCityDate(value);
  const now = new Date();
  const time = timeFmt.format(new Date(value));
  if (day === toCityDate(now)) return `сегодня, ${time}`;
  if (day === toCityDate(new Date(now.getTime() - 86_400_000)))
    return `вчера, ${time}`;
  return `${dayFmt.format(new Date(value))}, ${time}`;
};

const CommentItem = ({
  comment,
  canReply,
  onReply,
  eventId,
  isMine,
  authenticated,
  onDelete,
}: {
  comment: EventComment;
  eventId: string;
  isMine: boolean;
  authenticated: boolean;
  onDelete: () => void;
  canReply: boolean;
  onReply: () => void;
}) => {
  const hidden = comment.status === 'hidden' || comment.status === 'deleted';
  const meta = [comment.organizationName, formatMeta(comment.createdAt)]
    .filter(Boolean)
    .join(' · ');

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={TRANSITION.base}
      className={cn(
        'flex gap-3 rounded-[18px] px-4 py-3.5',
        comment.isOrganizer
          ? 'border-[1.5px] border-blue bg-blue-soft text-ink'
          : 'bg-background',
        comment.parentId && 'ml-6 sm:ml-10',
      )}
    >
      <span
        aria-hidden="true"
        className={cn(
          'grid size-10 shrink-0 place-items-center rounded-full font-extrabold',
          comment.isOrganizer ? 'bg-blue text-white' : 'bg-gold text-ink',
        )}
      >
        {initialOf(comment.author.name)}
      </span>
      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2 text-[15px]">
          <b>{comment.author.name}</b>
          {comment.isOrganizer && (
            <Badge className="px-2.5 py-0.5 text-[13px] font-bold">
              Организатор
            </Badge>
          )}
          <span className="text-muted-foreground">{meta}</span>
        </div>
        {hidden ? (
          <i className="text-muted-foreground">
            {comment.status === 'deleted'
              ? 'Сообщение удалено автором'
              : 'Сообщение скрыто модератором'}
          </i>
        ) : (
          <span className="break-words whitespace-pre-line">
            {comment.body}
          </span>
        )}
        {canReply && !hidden && (
          <Button
            variant="link"
            size="sm"
            className="self-start"
            onClick={onReply}
          >
            Ответить
          </Button>
        )}
        {authenticated && comment.status === 'visible' && (
          <div className="flex flex-wrap gap-2">
            {isMine ? (
              <Button variant="link" size="sm" onClick={onDelete}>
                Удалить своё сообщение
              </Button>
            ) : (
              <ReportButton eventId={eventId} commentId={comment.id} />
            )}
          </div>
        )}
      </div>
    </motion.div>
  );
};

interface DiscussionProps {
  eventId: string;
  isOrganizer: boolean;
  /** Обсуждение закрыто: встреча снята, отменена или прошла. */
  closed?: boolean;
  draft: string;
  onDraftChange: (value: string) => void;
}

/** «Обсуждение»: вопрос можно задать до записи, ответ организатора выделен. */
export const Discussion = ({
  eventId,
  isOrganizer,
  closed,
  draft,
  onDraftChange,
}: DiscussionProps) => {
  const { data: comments, isPending, isError, refetch } = useComments(eventId);
  const add = useAddComment();
  const { data: user } = useCurrentUser();
  const cache = useQueryClient();
  const confirm = useConfirm();
  const remove = useMutation({
    mutationFn: (id: string) => deleteComment(eventId, id),
    onSuccess: () =>
      cache.invalidateQueries({ queryKey: COMMENT_KEYS.LIST(eventId) }),
  });
  const [replyTo, setReplyTo] = useState<EventComment | null>(null);
  const commentForm = useForm<CommentValues>({
    resolver: zodResolver(commentSchema),
    defaultValues: { body: draft },
  });
  const { setValue } = commentForm;
  useEffect(() => {
    setValue('body', draft);
  }, [draft, setValue]);

  // Ответы показываем сразу под вопросом.
  const roots = (comments ?? []).filter((c) => !c.parentId);
  const thread = roots.flatMap((root) => [
    root,
    ...(comments ?? []).filter((c) => c.parentId === root.id),
  ]);

  const send = commentForm.handleSubmit(({ body }) => {
    add.mutate(
      { eventId, body, parentId: replyTo?.id },
      {
        onSuccess: () => {
          onDraftChange('');
          setReplyTo(null);
        },
      },
    );
  });

  const isReply = isOrganizer || !!replyTo;
  let error = '';
  if (commentForm.formState.errors.body)
    error = commentForm.formState.errors.body.message ?? 'Проверьте сообщение';
  else if (add.isError)
    error = 'Не удалось отправить. Текст сохранён — попробуйте ещё раз.';

  let fieldLabel = 'Ваш вопрос организатору';
  if (replyTo) fieldLabel = `Ответ для ${replyTo.author.name}`;
  else if (isOrganizer) fieldLabel = 'Сообщение участникам';

  let sendLabel = isReply ? 'Ответить' : 'Задать вопрос';
  if (add.isPending) sendLabel = 'Отправляем…';
  else if (add.isError) sendLabel = 'Отправить ещё раз';

  return (
    <div className="flex flex-col gap-3 rounded-3xl bg-card p-5">
      {isPending && (
        <div className="flex flex-col gap-3" aria-busy="true">
          <Skeleton className="h-16 rounded-[18px]" />
          <Skeleton className="ml-10 h-16 rounded-[18px]" />
        </div>
      )}
      {isError && (
        <div role="alert" className="flex flex-wrap items-center gap-3">
          <span>Не удалось загрузить обсуждение.</span>
          <Button variant="outline" size="sm" onClick={() => void refetch()}>
            Повторить
          </Button>
        </div>
      )}
      {!isPending && !isError && thread.length === 0 && (
        <div>
          <b>Вопросов пока нет.</b> Спросите — ответ увидят все, кто смотрит
          встречу.
        </div>
      )}
      {thread.map((comment) => (
        <CommentItem
          key={comment.id}
          comment={comment}
          eventId={eventId}
          isMine={user?.id === comment.author.id}
          authenticated={!!user}
          onDelete={() => {
            void confirm({
              title: 'Удалить сообщение?',
              description: 'Ответы останутся в обсуждении.',
              confirmText: 'Удалить',
              destructive: true,
            }).then((ok) => {
              if (ok && !remove.isPending) remove.mutate(comment.id);
            });
          }}
          canReply={!!user && !closed && !comment.parentId}
          onReply={() => {
            setReplyTo(comment);
            document.getElementById(DISCUSSION_INPUT_ID)?.focus();
          }}
        />
      ))}

      {!user ? (
        <Button
          variant="outline"
          render={
            <Link
              to={`${STATIC_ROUTES.LOGIN}?next=${encodeURIComponent(DYNAMIC_ROUTES.EVENT(eventId))}`}
            />
          }
        >
          Войти и спросить
        </Button>
      ) : null}
      {!!user && closed && (
        <span className="text-muted-foreground">
          Обсуждение закрыто: встреча больше не принимает участников.
        </span>
      )}
      {!!user && !closed && (
        <>
          <div className="mt-1.5 flex flex-wrap items-center justify-between gap-2">
            <Label htmlFor={DISCUSSION_INPUT_ID}>{fieldLabel}</Label>
            {replyTo && (
              <Button variant="link" size="sm" onClick={() => setReplyTo(null)}>
                Отменить ответ
              </Button>
            )}
          </div>
          <Textarea
            id={DISCUSSION_INPUT_ID}
            rows={3}
            maxLength={2000}
            value={draft}
            onChange={(e) => {
              commentForm.clearErrors('body');
              commentForm.setValue('body', e.target.value, {
                shouldDirty: true,
              });
              onDraftChange(e.target.value);
            }}
            aria-invalid={!!error}
            aria-describedby={error ? 'discussion-error' : undefined}
            placeholder="Например: нужен ли свой инвентарь?"
          />
          {error && (
            <span
              id="discussion-error"
              role="alert"
              className="font-semibold text-danger"
            >
              ! {error}
            </span>
          )}
          <div className="flex flex-wrap items-center gap-3">
            <Button
              variant="outline"
              disabled={add.isPending}
              onClick={() => void send()}
            >
              {sendLabel}
            </Button>
            {!isOrganizer && (
              <span className="text-[15px] text-muted-foreground">
                Вопрос не записывает вас на встречу
              </span>
            )}
          </div>
        </>
      )}
    </div>
  );
};
