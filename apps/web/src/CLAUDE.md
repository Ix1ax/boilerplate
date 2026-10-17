# CLAUDE.md

## Стек

React 19 (React Compiler включён) · TypeScript · Vite 8 · TanStack Query 5 ·
react-router 7 · axios · Tailwind v4 + shadcn (base-ui) · pnpm.

## Команды

```bash
pnpm dev        # дев-сервер
pnpm build      # tsc -b + vite build
pnpm lint       # eslint .
pnpm preview    # превью прод-сборки
```

Перед коммитом нетривиальных правок: `pnpm lint` и typecheck
(`npx tsc -p tsconfig.app.json --ignoreDeprecations 6.0 --noEmit` — флаг гасит
предсуществующий варнинг про `baseUrl`, это не твоя ошибка).

## Архитектура — Feature-Sliced Design

Слои сверху вниз, импорт разрешён **только вниз**:

```
app → pages → widgets → features → entities → shared
```

- `app/` — bootstrap, провайдеры, роутер, layouts. Единственный слой, который
  связывает `shared` с `entities` (инъекция зависимостей).
- `shared/` — переиспользуемое ядро без знания о домене. **`shared` НЕ импортирует
  из `entities`/`features`/выше.** Зависимости прокидываются через DI (см. ниже).
- Остальные слои — по фичам/страницам/сущностям.

### Публичный API слайсов

У каждого слайса есть баррел `index.ts`. **Импортируй из барреля слайса, а не из
глубоких путей:**

```ts
import { getAccessToken } from '@/entities/user'; // да
import { getAccessToken } from '@/entities/user/lib/token'; // нет (кроме app/bootstrap)
```

Алиас `@/` → `src/`. Новый публичный символ — добавь в `index.ts` слайса.

### Dependency Injection (ключевой паттерн)

`shared`-модули не знают про токены и домен. Конфиг инъектируется из `app/bootstrap.ts`:

- `shared/api` ← `configureApi({ getAccessToken, getRefreshToken, saveTokens, onUnauthorized })`
- `shared/ws` ← `configureWs({ getUrl, getToken, onUnauthorized })`

Добавляешь shared-модулю доменную зависимость — делай это тем же способом
(функция `configureX` + `getXConfig()`), а не импортом из `entities`.

## Стиль кода

- **Только arrow functions** — компоненты, хуки, утилиты. Генерики в `.tsx`
  пиши как `<T extends ...>(...)` (не `function`). Исключение — class-компоненты
  там, где нужен `componentDidCatch` (напр. `ErrorBoundary`).
- Prettier: 2 пробела, одинарные кавычки, точки с запятой (`.prettierrc`).
- **Порядок импортов** (ESLint `simple-import-sort`, автофикс, пустая строка между
  группами): внешние пакеты → слои FSD сверху вниз (`@/app`…`@/shared`) → прочий
  `@/` → относительные → side-effect (`import './globals.css'`) **последними**.
  Внутри слайса — импорт из сегментного барреля (`../api`, `../model`), deep-путь
  оставляем лишь чтобы разорвать цикл (тот же сегмент: `./constants`; обратное
  ребро `api → model`: `../model/types`).
- **Без вложенных тернарников** (`a ? b : c ? d : e`) — используй ранние `return`
  в хелпер-функции. Проверяется ESLint-правилом `no-nested-ternary`.
- Комментарии и пользовательские/лог-строки — на русском (как в кодовой базе).
- TS-конфиг строгий, учитывай ограничения:
  - `verbatimModuleSyntax` → разделяй `import type { ... }` и значения.
  - `erasableSyntaxOnly` → **нельзя** `enum`, `namespace` с рантаймом,
    parameter properties (`constructor(private x)`). Используй `as const`-объекты
    и явные поля класса.
  - `noUnusedLocals` / `noUnusedParameters` включены.

## React 19

- Контекст: `<Context value={...}>` напрямую (без `.Provider`), чтение через `use(Context)`.
- React Compiler включён (babel-plugin-react-compiler) — **не расставляй
  `useMemo`/`useCallback` ради перфа вручную**, компилятор делает это сам.
  `useCallback`/`useRef` оправданы только для референсной стабильности (напр.
  функция в `deps` или колбэк, передаваемый вовне).
- **Пропсы** — именованный `interface XxxProps { ... }` (со своими свойствами;
  расширение нативных пропсов — через `extends`, напр.
  `interface PageHeaderProps extends ComponentProps<'div'> { ... }`). Исключение:
  если props **не добавляют свойств** (чистый алиас, напр.
  `type MotionDivProps = HTMLMotionProps<'div'>`) — это `type`. **Всё остальное —
  `type`** (`interface` допустим ещё только для declaration merging: `Register`,
  `ImportMetaEnv`). Правило не выражается одним ESLint-правилом — держим ревью.
  **`children`** — слот для вложенного JSX; типизируется `children?: ReactNode`
  (импорт `type ReactNode` из `'react'`). Необязателен (`?`), т.к. компонент-обёртку
  можно отрендерить и пустой. У провайдеров/layout это обычно единственный проп.

## Данные и ошибки

- **TanStack Query** типизирован через `Register` (`src/app/types/tanstack-query.d.ts`):
  `defaultError = ApiError`, `meta.silent?: boolean` для подавления глобального тоста.
  Глобальный `queryClient` и `setErrorHandler` — в `shared/api`.
- **Ошибки**: `ApiError extends ErrorLogger` (`shared/api/errors.ts`,
  `shared/lib/errorLogger.ts`). Логгер — `logger` из `shared/lib` с методом
  `.child(scope)`; он редактит чувствительные поля (`token`, `password`, …).
- **axios** — синглтон `API` с интерсепторами (авто-refresh токена по 401,
  single-flight). HTTP-запросы через него.

## Запросы (TanStack Query)

Два уровня на слайс: **функция запроса** (`api/`) и **хук-обёртка** (`model/`).

- `api/` — «тупые» функции на `request<T>`: без React, без query keys, без
  инвалидации. Наружу слайса обычно не торчат (в баррель — только хуки). Пример:
  `entities/user/api/userApi.ts` (`getMe`, `getUser`, `getUsers`).
- `model/` — хук с `useQuery`/`useMutation`, который и держит ключи/инвалидацию.
  Пример: `useCurrentUser` (query), `useLogin`/`useLogout` (mutations).

Правила:

- **Query keys — фабрика-константа** рядом с сущностью, не строки по месту.
  Пример — `USER_KEYS` (`entities/user/model/constants.ts`):
  ```ts
  export const USER_KEYS = {
    ALL: ['users'] as const,
    ME: ['users', 'me'] as const,
    LIST: () => ['users', 'list'] as const,
    DETAIL: (id: string) => ['users', 'detail', id] as const,
  };
  ```
- **Queries — в `entities`** (чтение домена); **mutations-действия — в `features`**
  (логин, и т.п.). Инвалидация из мутации идёт по ключам сущности (`useLogin` →
  `invalidateQueries({ queryKey: USER_KEYS.ME })`).
- **Хук возвращает сырой результат `use*(...)` как есть** (`return useQuery(...)` /
  `return useMutation(...)`) — единый стиль для queries и mutations, не ломает
  render-tracking TanStack. Производные доменные значения — через `select`, а не
  рукодельный объект-обёртку. Потребитель читает `data` (`const { data: user } =
  useCurrentUser()`).
- Ошибки типизированы как `ApiError` (через `Register`); глобальный тост —
  `setErrorHandler`, подавить точечно — `meta: { silent: true }` на запросе.

```ts
// api/userApi.ts — функция запроса
export const getUser = (id: string) => request<User>({ url: `/users/${id}` });

// model/useUser.ts — хук-обёртка
export const useUser = (id: string) =>
  useQuery({ queryKey: USER_KEYS.DETAIL(id), queryFn: () => getUser(id) });
```

## WebSocket (`shared/ws`)

Единый нативный `WsClient` (reconnect с backoff+jitter, heartbeat, очередь
отправки, типизированный pub/sub). Протокол — в `types.ts`: расширяй юнионы
`WsIncoming`/`WsOutgoing`, остальной слой типизируется сам.

```ts
useWsStatus()                              // статус соединения
useWsSubscription('notification', handler) // подписка, авто-отписка
useWsSend()                                // отправка (очередь до open)
useWsInvalidate('event', ['queryKey'])     // realtime → инвалидация кэша
bindWsToQuery(wsClient, queryClient, {...}) // декларативный мост (вне хуков)
```

Настройки reconnect/heartbeat — в `WS_TUNING` (`config.ts`). Подключение —
`WsProvider` в `app/providers/Providers.tsx`; конфиг — `configureWs` в `app/bootstrap.ts`.

## UI-компоненты (`shared/ui`)

shadcn, стиль `base-nova` на `@base-ui/react` (не Radix), иконки `lucide-react`,
`cn` — из пакета `"cn"`. `components.json` настроен под FSD (`ui → @/shared/ui`,
css → `src/app/globals.css`). Токены темы — в `src/app/globals.css` (светлая/`.dark`).
Импорт — из барреля `@/shared/ui`.

Компоненты приведены к нашему стилю: **arrow-функции** (как и везде) и **явные
именованные экспорты** в барреле (`export { A, B } from "./X"`, без `export *`).
cva-варианты (`buttonVariants`, …) остаются рядом с компонентом — из-за них правило
`react-refresh/only-export-components` для `shared/ui` выключено в `eslint.config.js`.

**После `shadcn add <name>` (обязательно) — CLI генерит не в нашем стиле:**

1. Переименуй файл из kebab (`dropdown-menu.tsx`) в **PascalCase** (`DropdownMenu.tsx`);
   у нас все файлы компонентов с большой буквы.
2. Переведи `function X(...) {}` → `const X = (...) => {}` (arrow-стиль).
3. Поправь внутренние импорты на PascalCase (напр. `@/shared/ui/Button`) — иначе
   упадёт на регистрозависимой ФС (CI/Linux).
4. Добавь **явный** реэкспорт в `src/shared/ui/index.ts`:
   `export { <Name> } from "./<Name>"`.
5. Проверь затянутые CLI зависимости: `next-themes`/`sonner-native` не нужны (web),
   тосты — на `sonner` через `shared/ui` `<Toaster />` (уже в `Providers`).

## Хуки и утилиты

- `@/shared/hooks` — `useToggle`, `usePrevious`, `useEventListener`,
  `useOnClickOutside`, `useMediaQuery`, `useInterval`, `useTimeout`, `useDebounce`,
  `useMountedRef`, `useCopyToClipboard`. Один хук — один файл, реэкспорт из барреля.
- `@/shared/utils` — чистые функции без React: `sleep`, `isAbortError`, `clamp`,
  `safeJsonParse`, `groupBy`, `uniqueBy`. (`shared/lib` — для «тяжёлых» модулей:
  `logger`, `errorLogger`, `localStorage`, `cn`.)

Перед добавлением нового — проверь, нет ли уже готового, чтобы не плодить дубли.

## Формы

react-hook-form + zod (`@hookform/resolvers/zod`) + обёртка `Form` в `@/shared/ui`:
`Form`, `FormField`, `FormItem`, `FormLabel`, `FormControl`, `FormMessage`,
`FormDescription`. `FormControl` прокидывает `id`/`aria-*` в поле (`Input`/`Select`).
Схема — в `model/schema.ts` фичи, `type Values = z.infer<typeof schema>`.
Пример — `features/auth/ui/LoginForm.tsx`.

## Фичи

- **`features/auth`** — логин/логаут. `useLogin` (API → `setTokens` →
  инвалидация `USER_KEYS.ME` → redirect на `location.state.from`), `useLogout`
  (`meta.silent`, чистит токены+кэш в `onSettled`). Форма — `LoginForm`.
- **`features/theme`** — переключение темы (это действие пользователя → фича, не
  `shared`). `ThemeProvider` (класс `.dark` на `<html>` + `localStorage` +
  `prefers-color-scheme`), `useTheme`, `ThemeToggle`. Провайдер — в `Providers`.
- **`features/confirm`** — императивные подтверждения через промис
  (`const ok = await confirm({ title, destructive })`). `ConfirmProvider` (один
  `Dialog` на приложение) + `useConfirm()`. Провайдер — в `Providers`.

Провайдер + действие пользователя (тема, confirm, …) — это фича, а не `shared`.
`shared/ui` остаётся «глупым»: не зависит от фич.

## Анимации (`shared/motion`)

Библиотека `motion` (импорт из `motion/react`). **Токены — единый источник:**
`DURATION`/`EASING`/`SPRING` (`tokens.ts`) → пресеты `TRANSITION` (`transitions.ts`)
→ `Variants` (`variants.ts`). Не хардкодь длительности/кривые по месту — бери из
токенов/вариантов.

- `MotionProvider` (в `Providers`) — `reducedMotion="user"`, автоматически уважает
  `prefers-reduced-motion`; дефолтный `transition` для всех motion-компонентов.
- Примитивы: `FadeIn`, `SlideUp`, `ScaleIn`, `Stagger` + `StaggerItem` (списки).
- Новый эффект — добавляй `Variants` в `variants.ts` (на базе `TRANSITION`), а не
  inline-конфиг в компоненте.

## Прочее (shared)

- `@/shared/config` — `env` (zod-валидация `import.meta.env`, падает с понятным
  сообщением при отсутствии `VITE_API_URL`). HTTP/WS читают URL отсюда.
- `@/shared/ui` layout-примитивы: `Container`, `Section`, `PageHeader`.
