import path from 'node:path';
import babel from '@rolldown/plugin-babel';
import tailwindcss from '@tailwindcss/vite';
import react, { reactCompilerPreset } from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  resolve: { alias: { '@': path.resolve(import.meta.dirname, 'src') } },
  plugins: [
    react(),
    tailwindcss(),
    babel({ presets: [reactCompilerPreset()] }),
  ],
  server: {
    port: 3000,
    strictPort: true,
    proxy: {
      // ws: true — проксировать и WebSocket (/api/v1/ws).
      '/api': { target: 'http://localhost:8080', ws: true },
      '/auth': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
    },
  },
});
