import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  reportReasonLabels,
  reportSchema,
  type ReportValues,
} from '@starter/contracts';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';

import { request, toApiError } from '@/shared/api';
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Label,
  Notice,
  Textarea,
} from '@/shared/ui';

interface ReportButtonProps {
  eventId: string;
  commentId?: string;
}

export const ReportButton = ({ eventId, commentId }: ReportButtonProps) => {
  const [open, setOpen] = useState(false);
  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<z.input<typeof reportSchema>, unknown, ReportValues>({
    resolver: zodResolver(reportSchema),
    defaultValues: { reason: 'wrong_info', details: '' },
  });
  const reason = useWatch({control,name:'reason'});
  const report = useMutation({
    mutationFn: (data: ReportValues) =>
      request({
        url: `/events/${eventId}${commentId ? `/comments/${commentId}` : ''}/report`,
        method: 'POST',
        data,
      }),
    meta: { silent: true },
  });
  return (
    <>
      <Button
        variant="link"
        size="sm"
        onClick={() => {
          report.reset();
          setOpen(true);
        }}
      >
        Пожаловаться
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Сообщить о нарушении</DialogTitle>
            <DialogDescription>
              Жалобу проверит модератор. Автор не увидит ваше имя.
            </DialogDescription>
          </DialogHeader>
          {report.isSuccess ? (
            <Notice tone="success" role="status">
              Жалоба отправлена. Решение придёт в уведомления.
            </Notice>
          ) : (
            <form
              className="grid gap-4"
              noValidate
              onSubmit={(e) =>
                void handleSubmit((values) => report.mutate(values))(e)
              }
            >
              <Label htmlFor={`report-reason-${commentId ?? eventId}`}>
                Причина
              </Label>
              <select
                id={`report-reason-${commentId ?? eventId}`}
                className="min-h-12 rounded-lg border border-input bg-card px-3"
                {...register('reason')}
              >
                {Object.entries(reportReasonLabels).map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
              <Label htmlFor={`report-details-${commentId ?? eventId}`}>
                Подробности{' '}
                {reason === 'other' ? '(обязательно)' : '(необязательно)'}
              </Label>
              <Textarea
                id={`report-details-${commentId ?? eventId}`}
                maxLength={2000}
                {...register('details')}
                aria-invalid={!!errors.details}
              />
              {errors.details && (
                <Notice tone="danger" role="alert">
                  {errors.details.message}
                </Notice>
              )}
              {report.isError && (
                <Notice tone="danger" role="alert">
                  {toApiError(report.error).message}
                </Notice>
              )}
              <Button type="submit" disabled={report.isPending}>
                {report.isPending ? 'Отправляем…' : 'Отправить жалобу'}
              </Button>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
};
