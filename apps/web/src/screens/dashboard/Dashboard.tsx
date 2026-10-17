'use client';
import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import type { TaskPage, User } from '@starter/contracts';
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Checkbox,
} from '@starter/ui';
import { useTheme } from 'next-themes';
import { tasksApi } from '@/entities/task';
import { CreateTaskForm } from '@/features/create-task/CreateTaskForm';
import { UploadFile } from '@/features/upload-file/UploadFile';
import { api, ApiError } from '@/shared/api/client';
export const Dashboard = () => {
  const [user, setUser] = useState<User | null>(null);
  const [page, setPage] = useState<TaskPage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [offset, setOffset] = useState(0);
  const [revision, setRevision] = useState(0);
  const { resolvedTheme, setTheme } = useTheme();
  const reload = useCallback(() => setRevision((value) => value + 1), []);
  useEffect(() => {
    const controller = new AbortController();
    void Promise.all([
      api<User>('/me', { signal: controller.signal }),
      tasksApi.list(offset, controller.signal),
    ])
      .then(([identity, tasks]) => {
        if (!controller.signal.aborted) {
          setUser(identity);
          setPage(tasks);
          setError('');
        }
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        if (error instanceof ApiError && error.status === 401) {
          setUser(null);
          setPage(null);
          return;
        }
        setError(error instanceof Error ? error.message : 'Сервис недоступен');
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [offset, revision]);
  const mutate = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError('');
    try {
      await action();
      reload();
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Ошибка запроса');
    } finally {
      setBusy(false);
    }
  };
  return (
    <main className="mx-auto max-w-5xl space-y-8 px-4 py-10 sm:px-8">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <Link className="font-semibold tracking-tight" href="/">
          Hackathon starter <Badge variant="secondary">0.1</Badge>
        </Link>
        <div className="flex gap-2">
          <Button
            variant="ghost"
            onClick={() =>
              setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')
            }
          >
            Сменить тему
          </Button>
          {user ? (
            <Button
              variant="outline"
              onClick={() =>
                void mutate(async () => {
                  const result = await fetch('/api/session/logout', {
                    method: 'POST',
                  });
                  if (!result.ok) throw new Error('Не удалось выйти');
                  const { url } = (await result.json()) as { url: string };
                  window.location.assign(url);
                })
              }
            >
              Выйти
            </Button>
          ) : (
            <Button asChild>
              <a href="/auth/login">Войти</a>
            </Button>
          )}
        </div>
      </header>
      <section className="space-y-3">
        <p className="text-sm text-muted-foreground">
          NEXT.JS · GO · POSTGRESQL
        </p>
        <h1 className="text-3xl font-semibold tracking-tight sm:text-5xl">
          Следующая идея начинается здесь.
        </h1>
        <p className="max-w-2xl text-muted-foreground">
          Готовая основа для команды: вход, API, база, UI и файлы. Меняйте
          предметную область и начинайте делать продукт.
        </p>
      </section>
      {error && (
        <div
          role="alert"
          className="rounded-lg border border-destructive p-4 text-sm"
        >
          {error}{' '}
          <Button variant="link" onClick={reload}>
            Повторить
          </Button>
        </div>
      )}
      {loading ? (
        <p role="status">Подключаемся…</p>
      ) : user ? (
        <>
          <p className="text-sm text-muted-foreground">
            Вы вошли как {user.name || user.email}. Данные доступны только
            вашему аккаунту.
          </p>
          <div className="grid items-start gap-6 md:grid-cols-[1.4fr_1fr]">
            <Card>
              <CardHeader>
                <CardTitle>Задачи</CardTitle>
                <CardDescription>
                  Рабочий пример полного цикла: форма → API → PostgreSQL.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <CreateTaskForm
                  onCreated={() => {
                    setOffset(0);
                    reload();
                  }}
                />
                <ul className="divide-y divide-border">
                  {page?.items.map((task) => (
                    <li key={task.id} className="flex items-center gap-3 py-3">
                      <Checkbox
                        aria-label={`Завершить: ${task.title}`}
                        checked={task.completed}
                        disabled={busy}
                        onCheckedChange={(checked) =>
                          void mutate(() =>
                            tasksApi.complete(task.id, checked === true),
                          )
                        }
                      />
                      <span
                        className={`min-w-0 flex-1 break-words text-sm ${task.completed ? 'text-muted-foreground line-through' : ''}`}
                      >
                        {task.title}
                      </span>
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={busy}
                        aria-label={`Удалить: ${task.title}`}
                        onClick={() =>
                          void mutate(async () => {
                            await tasksApi.remove(task.id);
                            if (page.items.length === 1 && offset > 0)
                              setOffset(Math.max(0, offset - 10));
                          })
                        }
                      >
                        Удалить
                      </Button>
                    </li>
                  ))}
                </ul>
                {page?.total === 0 && (
                  <p className="text-sm text-muted-foreground">
                    Создайте первую задачу.
                  </p>
                )}
                {page && page.total > 10 && (
                  <nav
                    aria-label="Страницы задач"
                    className="flex items-center gap-3"
                  >
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={offset === 0}
                      onClick={() => setOffset(Math.max(0, offset - 10))}
                    >
                      Назад
                    </Button>
                    <span className="text-sm">
                      {offset + 1}–{Math.min(offset + 10, page.total)} из{' '}
                      {page.total}
                    </span>
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={offset + 10 >= page.total}
                      onClick={() => setOffset(offset + 10)}
                    >
                      Далее
                    </Button>
                  </nav>
                )}
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>Файлы</CardTitle>
                <CardDescription>
                  Приватное S3-хранилище с проверкой владельца.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <UploadFile />
              </CardContent>
            </Card>
          </div>
        </>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Начните со входа</CardTitle>
            <CardDescription>
              Создайте аккаунт в Keycloak или используйте локального
              демопользователя из .env.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild>
              <a href="/auth/login">Войти или зарегистрироваться</a>
            </Button>
          </CardContent>
        </Card>
      )}
      <footer className="border-t pt-5 text-sm text-muted-foreground">
        Смените название, цвета и этот пример под свой следующий хакатон.
      </footer>
    </main>
  );
};
