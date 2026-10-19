import { useState } from 'react';

import { STATIC_ROUTES } from '@/shared/constants';
import { Button, Checkbox, Tabs, TabsList, TabsTrigger } from '@/shared/ui';

import { useLogin, useLoginTarget } from '../model';

/**
 * Вход и регистрация идут через Keycloak: пароль вводится на его странице и не попадает
 * в приложение. Здесь — выбор и пояснение, что будет дальше.
 */
export const LoginForm = () => {
  const login = useLogin();
  const target = useLoginTarget();
  const [tab, setTab] = useState<'login' | 'register'>('login');
  const [club, setClub] = useState(false);
  const [sent, setSent] = useState(false);

  const go = (next?: string) => {
    setSent(true);
    login(next);
  };

  return (
    <div className="flex flex-col gap-5">
      <Tabs value={tab} onValueChange={(value) => setTab(value)}>
        <TabsList
          aria-label="Вход или регистрация"
          className="grid w-full grid-cols-2"
        >
          <TabsTrigger value="login" className="min-h-12 text-[17px]">
            Вход
          </TabsTrigger>
          <TabsTrigger value="register" className="min-h-12 text-[17px]">
            Регистрация
          </TabsTrigger>
        </TabsList>
      </Tabs>

      <div className="flex flex-col gap-[18px] rounded-3xl bg-card p-7">
        {tab === 'login' ? (
          <>
            <p className="text-lg">
              Войдите по логину или email. Откроется общая страница входа —
              после неё вернём вас сюда.
            </p>
            <Button size="lg" disabled={sent} onClick={() => go()}>
              {sent ? 'Переходим ко входу…' : 'Войти'}
            </Button>
          </>
        ) : (
          <>
            <p className="text-lg">
              Аккаунт нужен, чтобы записываться, задавать вопросы и создавать
              встречи. На странице входа выберите «Регистрация».
            </p>
            <label
              className={`flex items-start gap-3 rounded-2xl border-[1.5px] p-4 ${club ? 'border-blue bg-blue-soft text-ink' : 'border-line-strong dark:border-border'}`}
            >
              <Checkbox
                checked={club}
                onCheckedChange={setClub}
                className="mt-0.5"
              />
              <span className="flex flex-col">
                <b>Я представляю клуб или сообщество</b>
                <span className="text-[15px]">
                  После регистрации откроем заявку клуба. Публиковать от его
                  имени можно после одобрения.
                </span>
              </span>
            </label>
            <Button
              size="lg"
              disabled={sent}
              onClick={() => go(club ? STATIC_ROUTES.CLUB : target)}
            >
              {sent ? 'Переходим к регистрации…' : 'Создать аккаунт'}
            </Button>
          </>
        )}
        <span className="text-[15px] text-muted-foreground">
          Вход через общую систему учётных записей (Keycloak). Пароль не
          хранится в браузере.
        </span>
      </div>
    </div>
  );
};
