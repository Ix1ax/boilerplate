import { type ReactNode, useCallback, useEffect, useState } from 'react';

import { STORAGE_KEYS } from '@/shared/constants';
import { useMediaQuery } from '@/shared/hooks';
import { readLocalStorage, writeLocalStorage } from '@/shared/lib';

import { ThemeContext } from './context';
import type { ResolvedTheme, Theme } from './types';

interface ThemeProviderProps {
  children?: ReactNode;
  defaultTheme?: Theme;
}

const isTheme = (value: string | null): value is Theme =>
  value === 'light' || value === 'dark' || value === 'system';

const resolveTheme = (theme: Theme, systemDark: boolean): ResolvedTheme => {
  if (theme !== 'system') return theme;
  return systemDark ? 'dark' : 'light';
};

export const ThemeProvider = ({
  children,
  defaultTheme = 'system',
}: ThemeProviderProps) => {
  const [theme, setThemeState] = useState<Theme>(() => {
    const stored = readLocalStorage(STORAGE_KEYS.THEME);
    return isTheme(stored) ? stored : defaultTheme;
  });

  const systemDark = useMediaQuery('(prefers-color-scheme: dark)');
  const resolvedTheme = resolveTheme(theme, systemDark);

  useEffect(() => {
    document.documentElement.classList.toggle('dark', resolvedTheme === 'dark');
  }, [resolvedTheme]);

  const setTheme = useCallback((next: Theme) => {
    setThemeState(next);
    writeLocalStorage(STORAGE_KEYS.THEME, next);
  }, []);

  return (
    <ThemeContext value={{ theme, resolvedTheme, setTheme }}>
      {children}
    </ThemeContext>
  );
};
