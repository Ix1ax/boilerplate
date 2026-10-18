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
  showWidgets?: boolean;
}

export const Providers = ({ children, showWidgets = true }: ProvidersProps) => {
  return (
    <ThemeProvider>
      <AccessibleMotionProvider>
        <QueryClientProvider client={queryClient}>
          <ConfirmProvider>{children}</ConfirmProvider>
          {showWidgets && (
            <>
              <ThemeToaster />
              <Accessibility />
            </>
          )}
        </QueryClientProvider>
      </AccessibleMotionProvider>
    </ThemeProvider>
  );
};
