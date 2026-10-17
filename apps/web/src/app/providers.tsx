'use client';
import type { ReactNode } from 'react';
import { ThemeProvider } from 'next-themes';
import { Toaster } from 'sonner';
export const Providers = ({ children }: { children: ReactNode }) => (
  <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
    <>
      {children}
      <Toaster richColors closeButton />
    </>
  </ThemeProvider>
);
