# Разработка

## Новый предметный модуль

1. Добавьте SQL-миграцию с секциями `-- +goose Up` и `-- +goose Down`.
2. Новый домен — отдельный сервис. Скопируйте `apps/tasks` как шаблон (свой `go.mod`, база, миграции, Dockerfile), замените домен.
3. Добавьте сервис в `compose.yaml` (с `<имя>-migrate`), адрес в `Upstreams` gateway и маршрут-прокси в `apps/gateway/internal/httpapi`.
4. Ограничьте SQL владельцем или членством в команде. UUID не даёт права доступа.
5. Согласованно обновите OpenAPI, типы и Zod-схемы в `packages/contracts`.
6. Добавьте чтение в `entities`, действия в `features`, страницы в `pages`.
7. Проверьте сценарий, ошибки и попытку доступа из другого аккаунта.

## Команды

Используйте npm из корня: один `package-lock.json` для всех workspaces.

```sh
npm install имя-пакета -w @starter/web
make check
make integration
make smoke
```

`make integration` требует `.env`, PostgreSQL и Redis: проверяет миграции, владельцев, пагинацию и полный цикл браузерной сессии с тестовым OIDC-сервером. Smoke требует работающий стек и двух демопользователей. Клиент `hackathon-cli` с password grant предназначен только для локальной проверки API, на сервере его отключают.

Для Go: у каждого сервиса свой модуль, команды запускаются в его каталоге.

```sh
cd apps/tasks   # или apps/gateway, apps/files
go get путь/пакета
go mod tidy
```

Инфраструктура (`make infra`): `tasks-postgres` на :5433, `files-postgres` на :5434, `rabbitmq` на :5672 (консоль http://localhost:15672, логин и пароль в `.env`). Запуск сервисов вне Docker (после `make infra`): `make gateway`, `make tasks`, `make files`; tasks слушает :8081, files — :8082 (в `.env` для этого заданы `TASKS_URL` и `FILES_URL`).

Миграции есть у каждого сервиса и применяются к его базе: `make migrate`, состояние: `make migrate-status`. Добавляйте новую миграцию вместо редактирования применённой. Откат (`go run ./cmd/tasks migrate down` после загрузки `.env` с `DATABASE_URL=$TASKS_DATABASE_URL`) может удалить данные.

## Фронтенд

Vite, React Router, React Query, React Hook Form, Zod и Tailwind v4. `@/` указывает на `src`. Импорты только вниз по `app → pages → widgets → features → entities → shared`; ESLint проверяет алиасные импорты между слоями. Схема создания задачи и DTO берутся из `@starter/contracts`.

Основной UI находится в `shared/ui` на Base UI, тема — `src/app/globals.css`. Независимый `@starter/ui` на Radix сохранён как дополнительный набор. Сообщения и документация — по-русски. Ошибки загрузки показывайте с повтором, серверные ошибки читайте из `error.message`.

Браузер не читает токены: запросы идут на `/api/v1` с cookie, вход — навигацией на `/auth/login`. Пользователь загружается из `/me`, отсутствие сессии возвращает null в query. Пользовательские данные React Query очищаются при успешном выходе, затем происходит полная навигация.

## Конфигурация

| Файл                                   | Назначение                   |
| -------------------------------------- | ---------------------------- |
| `.env.example`                         | Корневой шаблон без секретов |
| `.env`                                 | Compose и Go через Makefile  |
| `web/.env.example`                     | Публичные настройки Vite     |
| `web/.env.local`                       | Локальные настройки Vite     |
| `.local/keycloak/hackathon-realm.json` | Первый импорт Keycloak       |

`make setup` создаёт отсутствующие настройки и сохраняет существующие значения. Пароли генерируются hex-строками, подходящими для DATABASE_URL; `.env` имеет права 0600. При ручной смене пароля учитывайте URL-кодирование и shell-синтаксис: Makefile загружает `.env` как shell-файл.

`VITE_*` попадает в браузерную сборку: секреты там запрещены. API_URL по умолчанию `/api/v1`, WS_URL пустой: сервер WebSocket пока отсутствует. OIDC-переменные и APP_ORIGIN читает Go, в Docker они задаются окружением API.

Настройки realm импортируются только в пустую базу Keycloak. Повторный setup не меняет сохранённый realm и пароли существующих пользователей. После смены порта приложения обновите в существующем клиенте `hackathon-web` поля Valid redirect URIs (`http://localhost:3000/auth/callback`), Web origins (`http://localhost:3000`) и Valid post logout redirect URIs (`http://localhost:3000/`) через административную консоль Keycloak на `http://localhost:8180`. Удалять базу для смены адресов не нужно.

Перед показом проверьте браузером: вход, задачу, завершение, пагинацию, файл, перезагрузку, выход и вход другим пользователем. Проверьте также сетевую ошибку, истечение сессии и недоступность чужих записей.
