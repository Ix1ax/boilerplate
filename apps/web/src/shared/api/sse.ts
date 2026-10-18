import { API_BASE_URL } from './config';
import { ApiError } from './errors';

type StreamOptions = {
  path: string;
  body: unknown;
  signal?: AbortSignal;
  onToken: (text: string) => void;
};

/**
 * POST + SSE через fetch: EventSource не умеет POST. Cookie сессии браузер
 * отправляет сам, а Origin для CSRF-проверки Go выставляется автоматически.
 */
export const streamSSE = async ({
  path,
  body,
  signal,
  onToken,
}: StreamOptions) => {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal,
  }).catch((cause: unknown) => {
    if (signal?.aborted) throw cause;
    throw new ApiError('Нет соединения с сервером', { cause });
  });

  if (!response.ok || !response.body) {
    const data = (await response.json().catch(() => null)) as {
      error?: { message?: string };
    } | null;
    throw new ApiError(data?.error?.message ?? 'Не удалось получить ответ', {
      status: response.status,
      data,
      url: path,
      method: 'POST',
    });
  }

  const reader = response.body.pipeThrough(new TextDecoderStream()).getReader();
  let buffer = '';
  for (;;) {
    const { done, value } = await reader.read();
    if (done) return;
    buffer += value;
    let end: number;
    while ((end = buffer.indexOf('\n\n')) !== -1) {
      const block = buffer.slice(0, end);
      buffer = buffer.slice(end + 2);
      const event = /^event: (.+)$/m.exec(block)?.[1] ?? 'message';
      const data = /^data: (.*)$/m.exec(block)?.[1] ?? '{}';
      if (event === 'done') return;
      if (event === 'error') {
        const message = (JSON.parse(data) as { message?: string }).message;
        throw new ApiError(message ?? 'Генерация прервана', { url: path });
      }
      const token = (JSON.parse(data) as { t?: string }).t;
      if (token) onToken(token);
    }
  }
};
