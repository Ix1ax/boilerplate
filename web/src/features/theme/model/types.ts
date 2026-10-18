export type Theme = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';

export type ThemeContextValue = {
  /** Выбор пользователя (включая 'system'). */
  theme: Theme;
  /** Фактическая тема после разрешения 'system'. */
  resolvedTheme: ResolvedTheme;
  setTheme: (theme: Theme) => void;
};
