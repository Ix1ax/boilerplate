import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { App } from './App.tsx';
import { bootstrap } from './bootstrap.ts';
import { registerServiceWorker } from './register-service-worker.ts';

import './styles/globals.css';

bootstrap();
registerServiceWorker();

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
