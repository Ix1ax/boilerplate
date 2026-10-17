# Фронтенд

Vite + React + TypeScript, React Router, React Query, Tailwind и Base UI. Команды запускаются из корня монорепы через npm workspaces: `make web`, `make check`.

Vite слушает 3000 и проксирует `/api` и `/auth` в Go на 8080. В Docker SPA раздаёт Nginx, он же проксирует запросы в API. Вход выполняется через Keycloak, сессиями и токенами управляет Go.

Структура: `app → pages → widgets → features → entities → shared`. Основной UI — `src/shared/ui`; схемы и DTO — `@starter/contracts`. Подробнее: [разработка](../../docs/development.md), [архитектура](../../docs/architecture.md).
