import { useForm } from 'react-hook-form';
import { Link } from 'react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { optionalReasonSchema, reasonSchema } from '@starter/contracts';
import type { z } from 'zod';

import { useConfirm } from '@/features/confirm';

import type { Participant } from '@/entities/event';

import { DYNAMIC_ROUTES } from '@/shared/constants';
import { Badge, Button, Label, Textarea } from '@/shared/ui';

import { participantLabels } from '../model/constants';

interface ParticipantRowProps {
  participant: Participant;
  busy: boolean;
  onDecision: (id: string, accept: boolean, reason: string) => void;
  onRemove: (id: string, reason: string) => void;
}

export const ParticipantRow = ({
  participant,
  busy,
  onDecision,
  onRemove,
}: ParticipantRowProps) => {
  const confirm = useConfirm();
  const schema =
    participant.status === 'pending' ? optionalReasonSchema : reasonSchema;
  const form = useForm<
    z.input<typeof schema>,
    unknown,
    z.output<typeof schema>
  >({
    resolver: zodResolver(schema),
    defaultValues: { reason: '' },
  });
  const reject = async () => {
    const ok = await confirm({
      title: 'Отклонить заявку?',
      description:
        'Участник получит уведомление. Повторная запись будет недоступна.',
      confirmText: 'Отклонить',
      destructive: true,
    });
    if (ok)
      onDecision(
        participant.id,
        false,
        (form.getValues('reason') ?? '').trim(),
      );
  };
  return (
    <li className="grid gap-3 rounded-3xl bg-card p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Link
          className="font-bold underline underline-offset-4"
          to={DYNAMIC_ROUTES.PERSON(participant.user.id)}
        >
          {participant.user.name}
        </Link>
        <Badge>{participantLabels[participant.status]}</Badge>
      </div>
      {participant.firstTime && <p>Впервые на такой встрече</p>}
      {participant.message && (
        <p className="whitespace-pre-line">{participant.message}</p>
      )}
      {participant.decisionReason && (
        <p>Причина решения: {participant.decisionReason}</p>
      )}
      {participant.status === 'pending' && (
        <>
          <Label htmlFor={`reason-${participant.id}`}>
            Комментарий к отказу (необязательно, увидит участник)
          </Label>
          <Textarea
            id={`reason-${participant.id}`}
            {...form.register('reason')}
            maxLength={1000}
          />
          {form.formState.errors.reason && (
            <p role="alert">{form.formState.errors.reason.message}</p>
          )}
          <div className="flex flex-wrap gap-3">
            <Button
              size="lg"
              disabled={busy}
              onClick={() => onDecision(participant.id, true, '')}
            >
              Принять заявку
            </Button>
            <Button
              variant="outline"
              disabled={busy}
              onClick={() => void form.handleSubmit(reject)()}
            >
              Отклонить
            </Button>
          </div>
        </>
      )}
      {['confirmed', 'waitlist'].includes(participant.status) && (
        <form
          className="grid gap-3"
          noValidate
          onSubmit={(e) =>
            void form.handleSubmit(async ({ reason }) => {
              const ok = await confirm({
                title: 'Исключить участника?',
                description:
                  'Он увидит причину и не сможет записаться повторно.',
                confirmText: 'Исключить',
                destructive: true,
              });
              if (ok) onRemove(participant.id, reason);
            })(e)
          }
        >
          <Label htmlFor={`remove-${participant.id}`}>Причина исключения</Label>
          <Textarea
            id={`remove-${participant.id}`}
            {...form.register('reason')}
            maxLength={1000}
          />
          {form.formState.errors.reason && (
            <p role="alert">{form.formState.errors.reason.message}</p>
          )}
          <Button type="submit" variant="outline" disabled={busy}>
            Исключить из состава
          </Button>
        </form>
      )}
    </li>
  );
};
