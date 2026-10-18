import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { type LlmPrompt, llmPromptSchema } from '@starter/contracts';

import {
  Button,
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
  Textarea,
} from '@/shared/ui';

import { useAssistant } from '../model';

/** Образец AI-функции: запрос → потоковый ответ локальной модели. Решение остаётся за человеком. */
export const AssistantChat = () => {
  const { text, status, error, send, stop } = useAssistant();
  const form = useForm<LlmPrompt>({
    resolver: zodResolver(llmPromptSchema),
    defaultValues: { prompt: '' },
  });
  const busy = status === 'streaming';

  return (
    <div className="space-y-4">
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit((values) => void send(values))}
          className="space-y-2"
        >
          <FormField
            control={form.control}
            name="prompt"
            render={({ field }) => (
              <FormItem>
                <FormControl>
                  <Textarea
                    {...field}
                    rows={4}
                    placeholder="Спросите ассистента…"
                    aria-label="Запрос к ассистенту"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className="flex gap-2">
            <Button type="submit" disabled={busy}>
              {busy ? 'Генерация…' : 'Отправить'}
            </Button>
            {busy && (
              <Button type="button" variant="outline" onClick={stop}>
                Остановить
              </Button>
            )}
          </div>
        </form>
      </Form>
      {error && (
        <p role="alert" className="text-destructive text-sm">
          {error}
        </p>
      )}
      {text && (
        <div
          aria-live="polite"
          className="bg-muted rounded-md p-4 text-sm whitespace-pre-wrap"
        >
          {text}
        </div>
      )}
    </div>
  );
};
