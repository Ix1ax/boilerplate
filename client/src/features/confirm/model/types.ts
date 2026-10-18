import type { ReactNode } from 'react';

export type ConfirmOptions = {
  title: ReactNode;
  description?: ReactNode;
  confirmText?: string;
  cancelText?: string;
  /** Красная кнопка подтверждения для деструктивных действий. */
  destructive?: boolean;
};
