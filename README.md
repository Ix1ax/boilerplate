# Стартовая монорепа для хакатонов

Самостоятельный шаблон: Vite + React + TypeScript, Go API, PostgreSQL, Redis, приватные файлы S3 и Keycloak. Вход, проверка токенов и прав выполняются в Go. Браузер использует HttpOnly cookie с идентификатором сессии; access и refresh token хранятся в Redis.

## Быстрый старт

Нужны Docker Desktop с Compose v2 и Node.js 24 из `.nvmrc`. Для разработки API на компьютере нужен Go версии из `apps/api/go.mod`.

```sh
make up
```

Команда создаёт `.env` и локальный realm, собирает приложения, применяет миграции и ждёт готовности сервисов. Откройте [приложение](http://localhost:3000), нажмите «Войти» и зарегистрируйтесь либо используйте `DEMO_EMAIL` и `DEMO_PASSWORD` из `.env`. Пароли генерируются отдельно для каждой копии.

| Сервис         | Адрес                        |
| -------------- | ---------------------------- |
| Приложение     | http://localhost:3000        |
| Go API         | http://localhost:8080        |
| Готовность API | http://localhost:8080/readyz |
| Keycloak       | http://localhost:8180        |
| Mailpit        | http://localhost:8025        |
| PostgreSQL     | localhost:5432               |
| Redis          | localhost:6379               |
| S3 API         | http://localhost:8333        |

Данные PostgreSQL и S3 сохраняются в Docker volumes. Redis хранит временные браузерные сессии: после его перезапуска потребуется войти снова. `make down` останавливает сервисы, сохраняя тома. Для двух копий шаблона одновременно поменяйте имя проекта **и** порты.

## Что включено

- Вход и регистрация Keycloak через Authorization Code + PKCE S256, одноразовый `state`, серверное обновление токенов, выход.
- Проверка JWT: подпись, issuer, audience и срок действия. Роли `user` и `admin`; пример проверки роли — `/api/v1/admin/ping`.
- Задачи и файлы с пагинацией. Чтение, изменение и удаление ограничены владельцем в SQL.
- Загрузка PNG, JPEG, WebP, PDF и текста до 10 МиБ. Тип проверяется по содержимому, скачивание доступно только владельцу.
- Redis ограничивает изменяющие запросы: 60 в минуту на пользователя. При недоступности Redis запись закрывается.
- React Query, React Hook Form, общая Zod-схема, светлая/тёмная тема, UI на Base UI, ошибки и повтор загрузки.
- Go-миграции Goose, Docker-сборки, Nginx для SPA и проксирования API, CI, unit- и интеграционные проверки.

## Структура

```text
apps/api/
  cmd/api/                    # Запуск API
  cmd/migrate/                # Миграции
  internal/auth/              # JWT, OIDC и серверные сессии
  internal/tasks/             # Пример предметного модуля
  internal/files/             # Приватные файлы и список метаданных
  internal/database/          # Встроенные SQL-миграции
  internal/platform/          # PostgreSQL, Redis и S3
apps/web/src/
  app/                        # Провайдеры, маршруты, layouts
  pages/                      # Страницы
  widgets/                    # Композиция UI, шапка
  features/                   # Формы и действия пользователя
  entities/                   # Модели и чтение данных
  shared/                     # HTTP, UI, хуки и утилиты
packages/contracts/           # OpenAPI, TypeScript-типы и Zod-схемы
packages/ui/                  # Независимый дополнительный набор UI-компонентов
infra/                        # PostgreSQL и шаблон realm Keycloak
deploy/                       # Пример Caddy
scripts/                      # Настройка, экспорт и smoke-проверка
```

Основной UI нового фронтенда находится в `apps/web/src/shared/ui` и использует Base UI. `packages/ui` — отдельный набор на Radix, он не импортирует приложение. При добавлении компонентов выбирайте один набор для конкретного экрана.

## Локальная разработка

```sh
make install
make infra
make migrate
```

В двух терминалах:

```sh
make api
```

```sh
make web
```

Vite слушает порт 3000 и проксирует `/api`, `/auth` и `/healthz` на Go API, порт 8080. Контейнеры `api` и `web` при этом должны быть остановлены. После Docker-варианта выполните `make down`, затем команды выше.

Корневой `.env` читают Compose и команды Go. Vite читает `apps/web/.env.local`; значение по умолчанию для API — `/api/v1`. `make setup` сохраняет существующие настройки и пароли. При переносе старого `.env.local` сверяйте его с `.env.example`; серверные OIDC-переменные нужны только Go.

## Команды

| Команда                                | Назначение                                                  |
| -------------------------------------- | ----------------------------------------------------------- |
| `make setup`                           | Создать локальные настройки и realm                         |
| `make install`                         | Установить npm- и Go-зависимости                            |
| `make up` / `make infra`               | Весь стек / только зависимости                              |
| `make down` / `make logs`              | Остановка / логи                                            |
| `make migrate` / `make migrate-status` | Миграции / состояние                                        |
| `make check`                           | Go-тесты с race, vet, сборки, TS, ESLint, Prettier и Vitest |
| `make integration`                     | Настоящие PostgreSQL и Redis: владельцы, пагинация, сессии  |
| `make smoke`                           | Работающий стек: JWT, права, задачи, файлы и CSRF           |
| `make fmt`                             | Форматирование                                              |
| `make backup`                          | Дамп базы приложения в `.local/backups/`                    |

`make check` требует установленных зависимостей. Для `make integration` нужны `.env`, PostgreSQL и Redis; для `make smoke` — весь стек. GitHub Actions запускает только деплой на сервер при push в `main`, включая слияние PR. Проверки запускаются локально. Настройка SSH и серверного скрипта описана в [инструкции деплоя](docs/deployment.md#автоматический-деплой).

## API и браузерные сессии

Контракт — [packages/contracts/openapi.yaml](packages/contracts/openapi.yaml). Типы и входные схемы — [packages/contracts/src/index.ts](packages/contracts/src/index.ts). Они поддерживаются вместе с Go вручную.

| Метод         | Путь                                | Назначение                                    |
| ------------- | ----------------------------------- | --------------------------------------------- |
| GET           | `/healthz`, `/readyz`               | Процесс API / зависимости                     |
| GET           | `/auth/login?next=/dashboard/tasks` | Начать вход через Keycloak                    |
| GET           | `/auth/callback`                    | Проверить state, обменять код, создать сессию |
| POST          | `/auth/logout`                      | Удалить сессию и получить URL выхода Keycloak |
| GET           | `/api/v1/me`                        | Пользователь и массив `roles`                 |
| GET           | `/api/v1/admin/ping`                | Пример проверки роли admin                    |
| GET, POST     | `/api/v1/tasks`                     | Список / создание задачи                      |
| PATCH, DELETE | `/api/v1/tasks/{id}`                | Изменение completed / удаление                |
| GET, POST     | `/api/v1/files`                     | Список / multipart-загрузка поля file         |
| GET, DELETE   | `/api/v1/files/{id}`                | Скачивание / удаление                         |

Списки принимают `limit=1..100` и `offset=0..100000`, возвращают `items`, `total`, `limit`, `offset`. Браузер обращается на собственный origin. Изменяющие cookie-запросы требуют точный `Origin=APP_ORIGIN`; внешние клиенты используют Bearer JWT. Ошибка: `{ "error": { "code": "...", "message": "...", "request_id": "..." } }`; `request_id` в теле необязателен, в заголовке `X-Request-ID` есть всегда.

## Новый проект

```sh
node scripts/export.mjs /путь/к/новому-проекту имя-проекта
cd /путь/к/новому-проекту
make up
```

Экспорт исключает Git, секреты, локальные данные, зависимости и сборки, не перезаписывает существующий каталог. Название проекта заменяется в metadata, стартовой странице, Compose и realm. Имена `@starter/*`, realm `hackathon` и Go-модуля остаются стабильными. Ничего не публикуется.

Для нового продукта замените `tasks`, страницы в `src/pages`, настройте палитру `src/app/globals.css` и добавьте свою миграцию. Дальше: [архитектура](docs/architecture.md), [разработка](docs/development.md), [подготовка сервера](docs/deployment.md), [план хакатона](docs/hackathon-start.md).

## Если запуск не удался

- Docker недоступен: запустите Docker Desktop.
- Порт занят: остановите другую копию либо согласованно измените Compose, `.env`, Vite proxy и redirect URI Keycloak.
- `Invalid redirect_uri`: проверьте `APP_ORIGIN` и клиент `hackathon-web`. Импорт realm выполняется только в новую базу; существующий realm меняйте через админку.
- Пароли после изменения `.env` не подходят: они уже сохранены в Keycloak/PostgreSQL; изменение env их не сбрасывает.
- Ошибка audience: mapper должен добавлять `hackathon-api` в `aud` access token.
- API не готов: `docker compose --profile app logs api migrate postgres redis s3 keycloak`.

Удаление томов не входит в обычную остановку. Результаты проверки этой версии — [docs/verification.md](docs/verification.md).
