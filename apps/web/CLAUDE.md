# Работа с фронтендом

Следуйте корневому AGENTS.md и docs/development.md. Стек: Vite + React + React Router, npm workspaces с единственным корневым package-lock.json.

Слои: app → pages → widgets → features → entities → shared. Импорты направлены вниз, используйте публичные index.ts слайсов. Shared не импортирует предметные модули; настройки передаются из app/bootstrap.ts.

Основной UI — Base UI в shared/ui, тема — app/styles/globals.css. Формы: React Hook Form и Zod; входные API-схемы и DTO импортируются из @starter/contracts. Компоненты и хуки — arrow-функции, типы импортируются через import type. Пользовательские строки, документация и поясняющие комментарии — по-русски.

Токены не сохраняются в браузере. Вход — переход на /auth/login, выход — POST /auth/logout с последующей навигацией на возвращённый URL. Go хранит сессии в Redis, проверяет JWT, роли, Origin и владельцев.

WebSocket-утилиты пока не подключены: серверного протокола нет. Проверки перед завершением: make check, для инфраструктуры make integration и make smoke.
