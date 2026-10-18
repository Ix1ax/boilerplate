import { z } from 'zod';

const schema = z.object({
  // Один origin работает и в Docker-сборке без локального .env.
  VITE_API_URL: z.string().min(1, 'не задан').default('/api/v1'),
  // Опционален: пусто → WebSocket просто не подключается.
  VITE_WS_URL: z.string().optional().default(''),
});

const parsed = schema.safeParse(import.meta.env);

if (!parsed.success) {
  const issues = parsed.error.issues
    .map((issue) => `  • ${issue.path.join('.')}: ${issue.message}`)
    .join('\n');
  throw new Error(
    `Некорректные переменные окружения (см. .env.example):\n${issues}`,
  );
}

export const env = parsed.data;
