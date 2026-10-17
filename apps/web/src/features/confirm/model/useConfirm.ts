import { use } from 'react';

import { ConfirmContext } from './context';

export const useConfirm = () => {
  const confirm = use(ConfirmContext);
  if (!confirm)
    throw new Error('useConfirm должен быть внутри <ConfirmProvider>');
  return confirm;
};
