import { MoonIcon, SunIcon } from 'lucide-react';

import { Button } from '@/shared/ui';

import { useTheme } from '../model';

export const ThemeToggle = () => {
  const { resolvedTheme, setTheme } = useTheme();
  const isDark = resolvedTheme === 'dark';

  return (
    <Button
      variant="ghost"
      size="icon"
      aria-label={isDark ? 'Светлая тема' : 'Тёмная тема'}
      onClick={() => setTheme(isDark ? 'light' : 'dark')}
    >
      {isDark ? <SunIcon /> : <MoonIcon />}
    </Button>
  );
};
