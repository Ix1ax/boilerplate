import { useState } from 'react';
import { Link } from 'react-router';

import type { Participant } from '@/entities/event';

import { DYNAMIC_ROUTES } from '@/shared/constants';
import { Badge, Button, Label, Textarea } from '@/shared/ui';

import { participantLabels } from '../model/constants';

interface ParticipantRowProps {
  participant: Participant;
  busy: boolean;
  onDecision: (id: string, accept: boolean, reason: string) => void;
}

export const ParticipantRow = ({
  participant,
  busy,
  onDecision,
}: ParticipantRowProps) => {
  const [reason, setReason] = useState('');
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
      {participant.firstTime && (
        <p className="text-sm text-muted-foreground">
          Впервые на такой встрече
        </p>
      )}
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
            maxLength={1000}
            value={reason}
            onChange={(event) => setReason(event.target.value)}
          />
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
              onClick={() => onDecision(participant.id, false, reason)}
            >
              Отклонить
            </Button>
          </div>
        </>
      )}
    </li>
  );
};
