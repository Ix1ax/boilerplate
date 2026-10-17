'use client';
import { Button } from '@starter/ui';
export default function ErrorPage({ reset }: { reset: () => void }) {
  return (
    <main className="mx-auto max-w-xl space-y-4 p-10">
      <h1 className="text-2xl font-semibold">Что-то пошло не так</h1>
      <p>Повторите попытку. Если ошибка остаётся, проверьте логи приложения.</p>
      <Button onClick={reset}>Попробовать снова</Button>
    </main>
  );
}
