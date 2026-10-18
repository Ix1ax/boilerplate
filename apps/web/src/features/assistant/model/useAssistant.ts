import { useCallback, useRef, useState } from 'react';
import { type LlmPrompt } from '@starter/contracts';

import { streamSSE, toApiError } from '@/shared/api';

type Status = 'idle' | 'streaming' | 'error';

/** Состояние потокового ответа модели. Повторная отправка отменяет предыдущий запрос. */
export const useAssistant = () => {
  const [text, setText] = useState('');
  const [status, setStatus] = useState<Status>('idle');
  const [error, setError] = useState<string | null>(null);
  const controller = useRef<AbortController | null>(null);

  const stop = useCallback(() => {
    controller.current?.abort();
    setStatus('idle');
  }, []);

  const send = useCallback(async (input: LlmPrompt) => {
    controller.current?.abort();
    const current = new AbortController();
    controller.current = current;
    setText('');
    setError(null);
    setStatus('streaming');
    try {
      await streamSSE({
        path: '/llm/stream',
        body: input,
        signal: current.signal,
        onToken: (token) => setText((prev) => prev + token),
      });
      setStatus('idle');
    } catch (cause) {
      if (current.signal.aborted) return;
      setError(toApiError(cause).message);
      setStatus('error');
    }
  }, []);

  return { text, status, error, send, stop };
};
