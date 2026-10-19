# Административное приложение

Следуйте корневому AGENTS.md. Самостоятельная SPA с Vite `base: /admin/` и Router `basename: /admin`. Слои: app → pages → widgets → features → entities → shared. Импорты из `client` запрещены; общий код размещаем в `packages`, UI — `@starter/ui`, DTO — `@starter/contracts`.

В разработке открывайте `http://localhost:3000/admin/` через прокси client, а не порт 3001: gateway разрешает Origin основного сайта. API и вход — абсолютные пути `/api/v1/*` и `/auth/*`. Сессия — HttpOnly cookie; токены в браузере не хранятся. Роль на клиенте управляет интерфейсом, реальные права проверяет Go.

Команды: `npm run dev:admin` (только admin), `npm run dev` (обе SPA), `npm run build:admin`; общая проверка — `make check`. Публичные настройки Vite не должны содержать секреты. PWA принадлежит client, admin свой service worker не регистрирует.

Хуки каждой сущности объединяем в `entities/<сущность>/model/hooks.ts`, включая хуки React Query. Отдельные `use*.ts` внутри entity не создаём, даже если хук пока один. Типы, ключи запросов и преобразования остаются в `types.ts`, `constants.ts` и `mappers.ts`. Хуки экспортируем через `model/index.ts` и публичный `index.ts` сущности; потребители импортируют их из публичного API слайса.
