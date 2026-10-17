import type { ReactNode } from 'react';
import { QueryClientProvider } from '@tanstack/react-query';

import { Accessibility } from '@/widgets/accessibility';
import { ThemeToaster } from '@/widgets/toaster';

import { ThemeProvider } from '@/features/theme';

import { queryClient } from '@/shared/api';

import { AccessibleMotionProvider } from './AccessibleMotionProvider';
import { ConfirmProvider } from './ConfirmProvider';

interface ProvidersProps {
  children?: ReactNode;
}

export const Providers = ({ children }: ProvidersProps) => {
  return (
    <ThemeProvider>
      <AccessibleMotionProvider>
        <QueryClientProvider client={queryClient}>
          <ConfirmProvider>{children}</ConfirmProvider>
          <ThemeToaster />
          <Accessibility />
        </QueryClientProvider>
      </AccessibleMotionProvider>
    </ThemeProvider>
  );
};
