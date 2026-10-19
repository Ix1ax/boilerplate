import path from 'node:path';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  base: '/admin/',
  resolve: { alias: { '@': path.resolve(import.meta.dirname, 'src') } },
  plugins: [react(), tailwindcss()],
  server: {
    port: Number(process.env.ADMIN_DEV_PORT ?? 3001),
    strictPort: true,
    // В браузере админка открывается через client:3000/admin/.
    // Так обе SPA используют одну сессию и один разрешённый Origin gateway.
    proxy: {
      '/api': { target: 'http://localhost:8080', ws: true },
      '/auth': 'http://localhost:8080',
    },
  },
});
