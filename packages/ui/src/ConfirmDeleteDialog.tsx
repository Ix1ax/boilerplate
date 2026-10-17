import type { ReactNode } from 'react';

import { Button } from './Button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './Dialog';

interface ConfirmDeleteDialogProps {
  children?: ReactNode;
  title: string;
  description: string;
  error?: string;
  busy: boolean;
  confirmLabel?: string;
  busyLabel?: string;
  onConfirm: () => Promise<boolean>;
  onClose: () => void;
}
export const ConfirmDeleteDialog = ({
  children,
  title,
  description,
  error,
  busy,
  confirmLabel = 'Удалить',
  busyLabel = 'Удаление…',
  onConfirm,
  onClose,
}: ConfirmDeleteDialogProps) => {
  const confirm = async () => {
    if (busy) return;
    if (await onConfirm()) onClose();
  };
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !busy) onClose();
      }}
    >
      <DialogContent showCloseButton={!busy}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        {children}
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={busy}
            onClick={onClose}
          >
            Отмена
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={busy}
            onClick={() => void confirm()}
          >
            {busy ? busyLabel : confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
