import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  localAuthFormSchema,
  type LocalAuthFormValues,
} from '@starter/contracts';
import { useQuery } from '@tanstack/react-query';

import { toApiError } from '@/shared/api';
import { STATIC_ROUTES } from '@/shared/constants';
import {
  Button,
  Checkbox,
  Input,
  Notice,
  Tabs,
  TabsList,
  TabsTrigger,
} from '@/shared/ui';

import { getAuthProviders, localLogin, localRegister } from '../api/authApi';
import { useLogin, useLoginTarget } from '../model';

export const LoginForm = () => {
  const login = useLogin();
  const target = useLoginTarget();
  const providers = useQuery({
    queryKey: ['auth-providers'],
    queryFn: getAuthProviders,
    meta: { silent: true },
  });
  const [tab, setTab] = useState('login');
  const [club, setClub] = useState(false);
  const [pending, setPending] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const form = useForm<LocalAuthFormValues>({
    resolver: zodResolver(localAuthFormSchema),
    defaultValues: { mode: 'login', name: '', email: '', password: '' },
  });
  const submit = form.handleSubmit(async (values) => {
    setPending(true);
    setError('');
    try {
      if (tab === 'login')
        await localLogin({ email: values.email, password: values.password });
      else
        await localRegister({
          name: values.name ?? '',
          email: values.email,
          password: values.password,
        });
      window.location.assign(
        tab === 'register' && club ? STATIC_ROUTES.CLUB : target,
      );
    } catch (cause) {
      setError(toApiError(cause).message);
      setPending(false);
    }
  });
  const submitLabel = tab === 'login' ? 'Войти' : 'Создать аккаунт';
  return (
    <div className="flex flex-col gap-5">
      <Tabs
        value={tab}
        onValueChange={(value) => {
          setTab(value);
          form.setValue('mode', value === 'register' ? 'register' : 'login');
          form.clearErrors();
          setError('');
        }}
      >
        <TabsList
          aria-label="Вход или регистрация"
          className="grid w-full grid-cols-2"
        >
          <TabsTrigger value="login">Вход</TabsTrigger>
          <TabsTrigger value="register">Регистрация</TabsTrigger>
        </TabsList>
      </Tabs>
      {providers.isPending && <p role="status">Проверяем способы входа…</p>}
      {providers.isError && (
        <Notice tone="danger" role="alert">
          Не удалось загрузить способы входа.{' '}
          <Button onClick={() => void providers.refetch()}>Повторить</Button>
        </Notice>
      )}
      {providers.data?.local.enabled && (
        <form
          onSubmit={(event) => void submit(event)}
          noValidate
          className="flex flex-col gap-5 rounded-3xl bg-card p-7"
        >
          {tab === 'register' && (
            <label className="grid gap-2">
              Имя
              <Input
                autoComplete="name"
                required
                maxLength={100}
                {...form.register('name')}
              />
              <span className="text-sm text-muted-foreground">
                Его увидят организаторы и участники обсуждения.
              </span>
            </label>
          )}
          <label className="grid gap-2">
            Email
            <Input
              type="email"
              autoComplete="email"
              required
              {...form.register('email')}
            />
          </label>
          <label className="grid gap-2">
            Пароль
            <Input
              type={showPassword ? 'text' : 'password'}
              autoComplete={
                tab === 'login' ? 'current-password' : 'new-password'
              }
              required
              {...form.register('password')}
            />
            {tab === 'register' && (
              <span className="text-sm text-muted-foreground">
                Не короче 8 символов.
              </span>
            )}
          </label>
          <Button
            type="button"
            variant="ghost"
            className="self-start"
            aria-pressed={showPassword}
            onClick={() => setShowPassword(!showPassword)}
          >
            {showPassword ? 'Скрыть пароль' : 'Показать пароль'}
          </Button>
          {tab === 'register' && (
            <label className="flex items-start gap-3 rounded-2xl border p-4">
              <Checkbox checked={club} onCheckedChange={setClub} />
              <span>
                Я представляю клуб или сообщество. После регистрации откроем
                заявку клуба.
              </span>
            </label>
          )}
          {Object.values(form.formState.errors).length > 0 && (
            <Notice tone="danger" role="alert">
              {Object.values(form.formState.errors)
                .map((issue) => issue.message)
                .join('. ')}
            </Notice>
          )}
          {error && (
            <Notice
              tone="danger"
              role="alert"
              tabIndex={-1}
              ref={(element) => element?.focus()}
            >
              {error}
            </Notice>
          )}
          <Button type="submit" size="lg" disabled={pending}>
            {pending ? 'Подождите…' : submitLabel}
          </Button>
        </form>
      )}
      {providers.data?.sso.enabled && (
        <div className="grid gap-3">
          <Button
            variant="outline"
            size="lg"
            disabled={pending}
            onClick={() =>
              login(tab === 'register' && club ? STATIC_ROUTES.CLUB : target)
            }
          >
            Войти через {providers.data.sso.title}
          </Button>
          <p className="text-sm text-muted-foreground">
            Общая система учётных записей. После входа вернём вас на нужную
            страницу.
          </p>
        </div>
      )}
    </div>
  );
};
