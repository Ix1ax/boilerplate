import { useId } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { reasonSchema } from '@starter/contracts';
import type { z } from 'zod';

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
  schema?: typeof reasonSchema;
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

/** Единая схема ограничивает причину так же, как доменные решения сервера. */
export const ReasonDialog = ({
  schema = reasonSchema,
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
  const id = useId();
  const form = useForm<
    z.input<typeof reasonSchema>,
    unknown,
    z.output<typeof reasonSchema>
  >({ resolver: zodResolver(schema), defaultValues: { reason: '' } });
  const close = (next: boolean) => {
    if (!next) form.reset();
    onOpenChange(next);
  };
  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && <DialogDescription>{description}</DialogDescription>}
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={form.handleSubmit((values) => onSubmit(values.reason))}
          noValidate
        >
          <Label htmlFor={id}>Причина</Label>
          <Textarea
            id={id}
            {...form.register('reason')}
            aria-invalid={!!form.formState.errors.reason}
            aria-describedby={
              form.formState.errors.reason ? `${id}-error` : undefined
            }
          />
          {form.formState.errors.reason && (
            <p id={`${id}-error`} role="alert" className="text-danger">
              {form.formState.errors.reason.message}
            </p>
          )}
          {error != null && (
            <Notice role="alert" tone="danger">
              {toApiError(error).message}
            </Notice>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => close(false)}
            >
              Отмена
            </Button>
            <Button
              type="submit"
              variant={destructive ? 'destructive' : 'default'}
              disabled={pending}
            >
              {pending ? 'Сохраняем…' : submitLabel}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
