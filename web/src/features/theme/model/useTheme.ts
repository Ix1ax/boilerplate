import { use } from 'react';

import { ThemeContext } from './context';

export const useTheme = () => {
  const context = use(ThemeContext);
  if (!context) throw new Error('useTheme должен быть внутри <ThemeProvider>');
  return context;
};
