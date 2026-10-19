import { useTheme } from '@/features/theme';

import { Toaster } from '@/shared/ui';

export const ThemeToaster = () => {
  const { resolvedTheme } = useTheme();

  return <Toaster theme={resolvedTheme} />;
};
