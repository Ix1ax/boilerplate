import type { ReactNode } from 'react';
import { QueryClientProvider } from '@tanstack/react-query';

import { ThemeProvider, useTheme } from '@/features/theme';

import { queryClient } from '@/shared/api';
import { MotionProvider } from '@/shared/motion';
import { Toaster } from '@/shared/ui';

import { ConfirmProvider } from './ConfirmProvider';

interface ProvidersProps {
  children?: ReactNode;
}

const ThemeToaster = () => {
  const { resolvedTheme } = useTheme();
  return <Toaster theme={resolvedTheme} />;
};

export const Providers = ({ children }: ProvidersProps) => {
  return (
    <ThemeProvider>
      <MotionProvider>
        <QueryClientProvider client={queryClient}>
          <ConfirmProvider>{children}</ConfirmProvider>
          <ThemeToaster />
        </QueryClientProvider>
      </MotionProvider>
    </ThemeProvider>
  );
};
