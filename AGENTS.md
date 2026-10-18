# AGENTS.md — рабочие правила для AI-агентов

Шаблон хакатонного проекта: микросервисный Go-бэкенд + React SPA + Keycloak + локальная LLM (Qwen в Ollama). Цель — за часы собрать **работающий, красиво показанный** продукт. Фронтенд: `client/CLAUDE.md`.

## Карта репозитория

| Путь                           | Что это                                                                                                                 |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------- |
| `server/gateway`               | Единственный публичный вход (:8080): Keycloak OIDC/PKCE, сессии в Redis, JWT, роли, CSRF, rate limit, прокси в сервисы  |
| `server/tasks`, `server/files` | Примеры доменных сервисов. **Шаблон для нового сервиса — `server/tasks`**                                               |
| `server/notifications`         | Уведомления пользователя: подписан на `notification.requested` в RabbitMQ, API колокольчика                             |
| `server/audit`                 | Журнал действий: gateway публикует `audit.recorded` о каждом изменяющем запросе, admin читает `GET /api/v1/admin/audit` |
| `server/realtime`              | WebSocket `GET /api/v1/ws`: каналы `user:<id>`, `room:<имя>`, `broadcast`; реплики связаны через RabbitMQ               |
| `server/llm`                   | Шлюз к Ollama: генерация, SSE-стрим, JSON-режим, лимиты                                                                 |
| `client`                       | Vite + React, слои `app→pages→widgets→features→entities→shared`                                                         |
| `packages/contracts`           | Общий OpenAPI (собирается `make openapi` из `server/*/api/openapi.yaml`), TS-типы, Zod-схемы                            |
| `compose.yaml`, `Makefile`     | Весь стек: `make up`; Qwen отдельно: `make llm`                                                                         |
| `docs/`                        | architecture, development, git-flow, `hackathon-playbook.md` (критерии жюри → план на 48 ч), `jury.md` (защита)         |

Каждый Go-модуль независим: свой `go.mod`, база, миграции, Dockerfile. Общего Go-кода нет — обвязку (`internal/platform`) копируем, а не выносим.

## Общие правила

- Документацию, поясняющие комментарии и сообщения интерфейса пишем по-русски. Комментарии объясняют причины и ограничения, а не пересказывают код.
- Не переносим сюда предметную область прошлых проектов, реальные адреса и секреты: используем `.env.example`, локальные значения создаёт `make setup`.
- Во фронтенде зависимости направлены вниз: `app → pages → widgets → features → entities → shared`. `packages/ui` не зависит от приложения.
- API, OpenAPI, TypeScript-типы и входные схемы меняем согласованно: маршрут описывается в `server/<сервис>/api/openapi.yaml` того же сервиса — понятно, с примерами и всеми кодами ошибок. Не публикуем, не разворачиваем и не отправляем изменения в remote без прямого поручения пользователя.

## Команды

```
make setup        # .env с паролями и INTERNAL_TOKEN (идемпотентно)
make up / down    # весь стек в Docker
make llm          # контейнер llm-ollama + модель Qwen (несколько ГБ, один раз)
make gateway|tasks|files|llm-svc|web   # запуск вне Docker после make infra
make test         # юнит-тесты Go + vitest
make check        # test + gofmt + vet + typecheck + lint + build
make integration  # тесты с реальными Postgres/Redis
make smoke        # сквозная проверка живого стека (API)
make seed         # демо-данные для двух демопользователей
make demo         # up + seed + ссылки: показать проект с нуля
make prefetch     # скачать всё заранее (SKIP_LLM=1 — без Qwen); делать до хакатона на хорошем интернете
make reset        # стереть данные и поднять чистое демо (YES=1 — без вопроса); модель сохраняется
make slides       # презентация из docs/presentation/ (Marp) → PDF
make loadtest     # k6: RPS и задержки → loadtest/result.json (LOAD_VUS, LOAD_DURATION)
make openapi      # общий OpenAPI из server/*/api/openapi.yaml + TS-типы + копия для Swagger
make e2e          # Playwright против живого стека (UI + ИИ-проверка)
```

**Definition of done:** `make check` зелёный; для изменений API/инфраструктуры ещё `make smoke`; контракты (OpenAPI/Zod/TS) обновлены вместе с кодом; документация затронутого места актуальна.

## Архитектурные инварианты (нарушать нельзя)

1. Браузер ходит **только в gateway**. Сервисы не публикуют порты и принимают запросы только с `X-Internal-Token` + `X-User-ID` (`platform.TrustGateway`). Владелец записи — всегда `platform.Owner(c)`, проверка владельца — **в SQL** (`WHERE owner_id=$1`). Роли приходят в `X-User-Roles`, почта — в `X-User-Email` (`platform.Email(ctx)`): админские маршруты закрывайте `platform.RequireRole("admin")` в сервисе **и** `auth.RequireRole` в gateway.
2. Роли и доступ проверяет Go, не клиент. Токены не хранятся в браузере.
3. Сервисы не импортируют друг друга и не читают чужие базы (у каждого свой Postgres). Связь — HTTP через gateway или события в RabbitMQ через transactional outbox (образец — `server/tasks/internal/broker`). Обработчики событий идемпотентны.
4. Миграции — только новыми файлами в `internal/database/migrations` сервиса; применённые не правим.
5. Секреты только из окружения; в git — лишь `.env.example`. Не добавляйте значения по умолчанию для секретов.
6. LLM — советник, не судья: ответы модели проходят валидацию, решения по людям принимает человек (human-in-the-loop). Факты берём из БД, модель их только формулирует.

## Образец ИИ-функции (копировать, а не изобретать)

`server/tasks/internal/copilot`: **правила** (детерминированно ставят флаги и вердикт) → **LLM** (опциональный совет, строго валидируемый JSON, текст пользователя передаётся как данные в `<task>`) → **человек решает** (`decision_required`). Сбой модели не ломает проверку: остаются правила. Дорогие эндпоинты, вызывающие модель, ограничиваются в gateway тем же лимитом `rate:llm` (10 в минуту). Потоковый чат — `client/src/features/assistant` + `shared/api/sse.ts`.

## Готовые заготовки

| Нужно                                     | Где образец                                                                                                                                                                                    |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Уведомить пользователя из любого сервиса  | Положить в outbox событие `notification.requested` `{title, body, link}` с `user_id` (см. `SaveModeration` в `server/tasks/internal/tasks/outbox.go`)                                          |
| Асинхронная обработка / связь сервисов    | `internal/broker` (одинаковый файл во всех сервисах): `Options{Queue, Keys, Outbox}`                                                                                                           |
| Отчёт в Excel                             | `platform.NewCSV` + маршрут под `RequireRole("admin")` (`server/tasks/internal/tasks/export.go`), кнопка — `pages/dashboard-admin`                                                             |
| ИИ-проверка с человеком в контуре         | `server/tasks/internal/copilot`                                                                                                                                                                |
| Потоковый чат с моделью                   | `features/assistant` + `shared/api/sse.ts`                                                                                                                                                     |
| Страница только для роли                  | `ProtectedRoute roles={['admin']}` + `roles` у вкладки в `DashboardLayout`                                                                                                                     |
| Статусы и разрешённые переходы            | `platform.NewWorkflow` (`server/tasks/internal/platform/workflow.go`): таблица переходов, `Check` → `ErrTransition` (409)                                                                      |
| Список с фильтрами, сортировкой и поиском | `platform.ParseList` + `ListSpec` (белый список колонок, значения — параметрами SQL)                                                                                                           |
| Фоновое задание по расписанию             | `platform.Every(ctx, db, log, name, interval, job)` — одна реплика за раз через advisory lock Postgres                                                                                         |
| Защита от двойной отправки                | Заголовок `Idempotency-Key` у изменяющих запросов (gateway, Redis, 24 ч)                                                                                                                       |
| Письмо пользователю                       | Поле `email` в `notification.requested` (адрес — `platform.Email(ctx)`); локально письма в Mailpit                                                                                             |
| Live-обновления в UI                      | SSE `GET /api/v1/notifications/stream` + `EventSource` в `features/notifications`                                                                                                              |
| Двусторонний realtime (чат, «кто онлайн») | `useChannel(channel)` из `shared/ws` + сервис `server/realtime`; права на каналы — `realtime.Policy`; сервер пишет в канал событием `realtime.publish`. Образец — страница «Лайв»              |
| Уведомления в мессенджер MAX              | `server/notifications/internal/maxbot` (официальный клиент `max-bot-api-client-go`): задайте `MAX_BOT_TOKEN` и `MAX_BOT_USERNAME`, кнопка «Получать уведомления в MAX» появится в колокольчике |
| История статусов с комментарием           | `Workflow.RequireComment(...)` + `platform.RecordTransition` / `platform.History` (таблица `status_history`, писать в одной транзакции со сменой статуса)                                      |
| Дашборд админа                            | `GET /api/v1/admin/tasks/stats` (`Repository.Stats`): итоги + ряд за 14 дней в одном снимке                                                                                                    |
| Файлы-вложения к записи                   | `POST /api/v1/files` с `entity_type`/`entity_id`, список — `?entity_type=&entity_id=`                                                                                                          |
| Кто что сделал                            | Журнал `server/audit` (включается вместе с `AMQP_URL` у gateway)                                                                                                                               |
| Демо-данные                               | `fixtures/*.json` и `fixtures/files/` → `make seed` (через API, с событиями и уведомлениями)                                                                                                   |
| Найти цепочку по логам                    | `request_id` идёт из gateway в сервисы, outbox и события (`platform.RequestID(ctx)`)                                                                                                           |

## Как добавить сервис (рецепт на 10 минут)

1. `cp -r server/tasks server/<имя>`; замените модуль в `go.mod` и импортах, домен — в `internal/<имя>`, миграцию — в `internal/database/migrations`.
2. `compose.yaml`: контейнеры `<имя>-postgres`, `<имя>-migrate` (`command: [migrate, up]`), `<имя>-service` (имя контейнера = имя сервиса) с `INTERNAL_TOKEN` и `DATABASE_URL`; пароль БД — `<ИМЯ>_DB_PASSWORD` в `.env.example` и `scripts/setup.mjs`.
3. `.env.example`, `scripts/setup.mjs` (если новые секреты), `go.work`, цели `Makefile`.
4. Gateway: адрес в `config` (`<ИМЯ>_URL`), поле в `httpapi.Upstreams`, маршрут `api.Any("/<имя>", …)` + `/*path`.
5. Опишите маршруты в `server/<имя>/api/openapi.yaml` (тест `api/spec_test.go` сверит их с кодом), выполните `make openapi`, затем `entities` → `features` → `pages` во фронтенде.
6. Тесты: сервис на фейках `Store`, интеграционный тест репозитория, строка в `scripts/smoke.mjs`.

## Как работать агенту

- Сначала читайте код рядом с задачей и образец (`server/tasks`), потом пишите. Не угадывайте имена и пути.
- Малыми шагами: изменение → `go build ./... && go test ./...` в затронутом модуле → дальше. `make check` — перед коммитом.
- Не раздувайте контекст: читайте файлы по частям, ищите `grep`-ом, не печатайте большие логи (`| tail`). Тяжёлые независимые куски отдавайте подагентам.
- Ошибки пользователю — по-русски и без внутренних деталей; в лог — с `request_id`.
- Не коммитьте и не пушьте без просьбы. Коммиты — Conventional Commits (`docs/git-flow.md`).

## Ловушки, на которых уже спотыкались

- `httputil.ReverseProxy.Rewrite`: заголовки удаляем в хендлере **до** установки новых, иначе стираются свои.
- Порты на хосте могут быть заняты (Postgres, Redis, RabbitMQ): меняются через `TASKS_DB_PORT`, `FILES_DB_PORT`, `RABBITMQ_PORT`.
- Первый `make up` долгий (образы, Keycloak до ~1 мин). Qwen не в основном стеке — `make llm`.
- `WriteTimeout` gateway 30 с: стримы (SSE) продлевают дедлайн явно; у nginx для `/api/v1/llm/` отключена буферизация.
- Старые данные общей базы `starter` и старые тома не переносятся; переименование контейнеров — `docker compose down --remove-orphans` перед первым `make up`.

## Что смотрит жюри

Работающее демо важнее объёма. Держите: `make up` с нуля за одну команду, демо-пользователи и seed, понятный сценарий на 3 минуты, `docs/jury.md` (проблема → решение → архитектура → метрики → риски), одно заметное AI-преимущество с человеком в контуре.
