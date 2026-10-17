import Link from 'next/link';
import { Button } from '@starter/ui';
export default function NotFound() {
  return (
    <main className="space-y-4 p-10">
      <h1 className="text-2xl">Страница не найдена</h1>
      <Button asChild>
        <Link href="/">На главную</Link>
      </Button>
    </main>
  );
}
