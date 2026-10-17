import { type ReactNode, useCallback, useState } from 'react';

import {
  type Confirm,
  ConfirmContext,
  type ConfirmOptions,
} from '@/features/confirm';

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui';

type Pending = {
  options: ConfirmOptions;
  resolve: (ok: boolean) => void;
};

interface ConfirmProviderProps {
  children?: ReactNode;
}

export const ConfirmProvider = ({ children }: ConfirmProviderProps) => {
  const [pending, setPending] = useState<Pending | null>(null);

  const confirm = useCallback<Confirm>(
    (options) =>
      new Promise<boolean>((resolve) => {
        setPending((prev) => {
          // Новый диалог отменяет предыдущий, чтобы его вызывающий код не ждал бесконечно.
          prev?.resolve(false);

          return { options, resolve };
        });
      }),
    [],
  );

  const settle = (ok: boolean) => {
    pending?.resolve(ok);
    setPending(null);
  };

  const options = pending?.options;

  return (
    <ConfirmContext value={confirm}>
      {children}
      <Dialog
        open={!!pending}
        onOpenChange={(open) => {
          if (!open) settle(false);
        }}
      >
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>{options?.title}</DialogTitle>
            {options?.description ? (
              <DialogDescription>{options.description}</DialogDescription>
            ) : null}
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => settle(false)}>
              {options?.cancelText ?? 'Отмена'}
            </Button>
            <Button
              variant={options?.destructive ? 'destructive' : 'default'}
              onClick={() => settle(true)}
            >
              {options?.confirmText ?? 'Подтвердить'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </ConfirmContext>
  );
};
