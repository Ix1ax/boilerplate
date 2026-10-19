# Работа с фронтендом

Следуйте корневому AGENTS.md и docs/development.md. Стек: Vite + React + React Router, npm workspaces с единственным корневым package-lock.json.

Client обслуживает `/`, независимая SPA admin — `/admin/`. Импорты из `admin` запрещены; общий код выносится в `packages`. Между приложениями переходите обычной ссылкой `<a>`, чтобы загрузить другой документ. `npm run dev` из корня запускает обе SPA, `npm run dev:client` — только client. Подробности — `docs/microfrontends.md`.

Слои: app → pages → widgets → features → entities → shared. Импорты направлены вниз, используйте публичные index.ts слайсов. Shared не импортирует предметные модули; настройки передаются из app/bootstrap.ts.

Основной UI — Base UI в shared/ui, тема — app/styles/globals.css. Формы: React Hook Form и Zod; входные API-схемы и DTO импортируются из @starter/contracts. Компоненты и хуки — arrow-функции, типы импортируются через import type. Пользовательские строки, документация и поясняющие комментарии — по-русски.

Непустые пропсы компонентов объявляем именованными `interface ComponentNameProps`, а не `type` или inline-объектами. Для типов пропсов с пустым телом это правило не применяется.

Токены не сохраняются в браузере. Вход — переход на /auth/login, выход — POST /auth/logout с последующей навигацией на возвращённый URL. Go хранит сессии в Redis, проверяет JWT, роли, Origin и владельцев.

WebSocket-утилиты пока не подключены: серверного протокола нет. Проверки перед завершением: make check, для инфраструктуры make integration и make smoke.

Хуки каждой сущности объединяем в `entities/<сущность>/model/hooks.ts`, включая хуки React Query. Отдельные `use*.ts` внутри entity не создаём, даже если хук пока один. Типы, ключи запросов и преобразования остаются в `types.ts`, `constants.ts` и `mappers.ts`. Хуки экспортируем через `model/index.ts` и публичный `index.ts` сущности; потребители импортируют их из публичного API слайса.
