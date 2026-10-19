import { useState } from 'react';

import { toApiError } from '@/shared/api';
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Label,
  Notice,
  Textarea,
} from '@/shared/ui';

interface ReasonDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  submitLabel: string;
  destructive?: boolean;
  pending: boolean;
  error: unknown;
  onSubmit: (reason: string) => void;
}

/** Диалог с обязательной причиной: отказ заявке, снятие события, отклонение жалобы. */
export const ReasonDialog = ({
  open,
  onOpenChange,
  title,
  description,
  submitLabel,
  destructive,
  pending,
  error,
  onSubmit,
}: ReasonDialogProps) => {
  const [reason, setReason] = useState('');
  const [touched, setTouched] = useState(false);
  const trimmed = reason.trim();
  const invalid = touched && trimmed.length === 0;

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          setReason('');
          setTouched(false);
        }
        onOpenChange(next);
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && (
            <DialogDescription>{description}</DialogDescription>
          )}
        </DialogHeader>
        <Label htmlFor="reason-dialog-text">Причина</Label>
        <Textarea
          id="reason-dialog-text"
          value={reason}
          maxLength={1000}
          aria-invalid={invalid}
          onChange={(event) => setReason(event.target.value)}
        />
        {invalid && (
          <p role="alert" className="text-danger">
            Укажите причину: её увидит другая сторона
          </p>
        )}
        {error != null && (
          <Notice role="alert" tone="danger">
            {toApiError(error).message}
          </Notice>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Отмена
          </Button>
          <Button
            variant={destructive ? 'destructive' : 'default'}
            disabled={pending}
            onClick={() => {
              if (!trimmed) {
                setTouched(true);
                return;
              }
              onSubmit(trimmed);
            }}
          >
            {submitLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
