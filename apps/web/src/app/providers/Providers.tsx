import type { ReactNode } from 'react';
import { QueryClientProvider } from '@tanstack/react-query';

import { ConfirmProvider } from '@/features/confirm';
import { ThemeProvider } from '@/features/theme';

import { queryClient } from '@/shared/api';
import { MotionProvider } from '@/shared/motion';
import { Toaster } from '@/shared/ui';
import { WsProvider } from '@/shared/ws';

interface ProvidersProps {
  children?: ReactNode;
}

export const Providers = ({ children }: ProvidersProps) => {
  return (
    <ThemeProvider>
      <MotionProvider>
        <QueryClientProvider client={queryClient}>
          <ConfirmProvider>
            <WsProvider>{children}</WsProvider>
          </ConfirmProvider>
          <Toaster />
        </QueryClientProvider>
      </MotionProvider>
    </ThemeProvider>
  );
};
