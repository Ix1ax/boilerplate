import { Link } from 'react-router';
import {
  BellIcon,
  BotIcon,
  ScrollTextIcon,
  ShieldCheckIcon,
} from 'lucide-react';

import { useCurrentUser } from '@/entities/user';

import { STATIC_ROUTES } from '@/shared/constants';
import { useSmoothScroll } from '@/shared/hooks';
import { Button, Card, CardContent, Container, Section } from '@/shared/ui';

const FEATURES = [
  {
    icon: ShieldCheckIcon,
    title: 'Вход через Keycloak',
    description:
      'OIDC/PKCE, сессии в Redis, роли и владение проверяются на сервере — токены не хранятся в браузере.',
  },
  {
    icon: BotIcon,
    title: 'ИИ-копилот с человеком в контуре',
    description:
      'Правила ставят флаги и вердикт детерминированно, модель добавляет совет — решение всегда принимает человек.',
  },
  {
    icon: BellIcon,
    title: 'Живые уведомления',
    description:
      'SSE в интерфейсе, письма и мессенджер MAX — обо всём важном узнаёте сразу.',
  },
  {
    icon: ScrollTextIcon,
    title: 'Журнал аудита',
    description:
      'Каждое изменяющее действие фиксируется и доступно администратору для разбора инцидентов.',
  },
];

export const HomePage = () => {
  const { data: user } = useCurrentUser();
  useSmoothScroll();

  return (
    <Container>
      <Section className="items-center py-16 text-center sm:py-24">
        <h1 className="max-w-2xl text-4xl font-semibold tracking-tight sm:text-5xl">
          Хакатонный стартер: от идеи до демо за часы
        </h1>
        <p className="max-w-xl text-muted-foreground">
          Микросервисный Go-бэкенд, React-интерфейс, Keycloak и локальная
          ИИ-модель — всё собрано заранее, чтобы вы могли сосредоточиться на
          своей идее.
        </p>
        <div className="flex gap-3">
          {user ? (
            <Button size="lg" render={<Link to={STATIC_ROUTES.DASHBOARD} />}>
              Открыть рабочую область
            </Button>
          ) : (
            <>
              <Button size="lg" render={<Link to={STATIC_ROUTES.LOGIN} />}>
                Войти
              </Button>
              <Button
                size="lg"
                variant="outline"
                render={<Link to={STATIC_ROUTES.DASHBOARD} />}
              >
                Посмотреть рабочую область
              </Button>
            </>
          )}
        </div>
      </Section>

      <Section className="py-12">
        <div className="grid gap-4 sm:grid-cols-2">
          {FEATURES.map(({ icon: Icon, title, description }) => (
            <Card key={title}>
              <CardContent className="flex gap-4">
                <Icon className="size-6 shrink-0 text-primary" />
                <div className="space-y-1">
                  <h2 className="font-medium">{title}</h2>
                  <p className="text-sm text-muted-foreground">{description}</p>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </Section>

      <Section className="items-center py-12 text-center">
        <h2 className="text-xl font-semibold">Готовы попробовать?</h2>
        <p className="max-w-md text-sm text-muted-foreground">
          Демо-пользователи и тестовые данные уже настроены — войдите и
          посмотрите, как работают задачи, файлы и ассистент.
        </p>
        <Button
          render={
            <Link to={user ? STATIC_ROUTES.DASHBOARD : STATIC_ROUTES.LOGIN} />
          }
        >
          {user ? 'Перейти в рабочую область' : 'Войти и попробовать'}
        </Button>
      </Section>
    </Container>
  );
};
