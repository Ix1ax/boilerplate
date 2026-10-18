import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import simpleImportSort from 'eslint-plugin-simple-import-sort';
import tseslint from 'typescript-eslint';
import { defineConfig, globalIgnores } from 'eslint/config';

// Порядок групп импортов (между группами — пустая строка):
// внешние пакеты → слои FSD сверху вниз → прочий @/ → относительные →
// side-effect (напр. './globals.css') — последними.
const IMPORT_GROUPS = [
  ['^node:', '^react', '^@?\\w'],
  ['^@/app(/|$)'],
  ['^@/pages(/|$)'],
  ['^@/widgets(/|$)'],
  ['^@/features(/|$)'],
  ['^@/entities(/|$)'],
  ['^@/shared(/|$)'],
  ['^@/'],
  ['^\\.'],
  ['^\\u0000'],
];

// FSD: импорт разрешён только вниз (app → pages → widgets → features →
// entities → shared). Для каждого слоя запрещаем импорт из вышестоящих.
const LAYERS_ABOVE = {
  shared: ['entities', 'features', 'widgets', 'pages', 'app'],
  entities: ['features', 'widgets', 'pages', 'app'],
  features: ['widgets', 'pages', 'app'],
  widgets: ['pages', 'app'],
  pages: ['app'],
};

const fsdBoundary = (layer, forbidden) => ({
  files: [`src/${layer}/**/*.{ts,tsx}`],
  rules: {
    'no-restricted-imports': [
      'error',
      {
        patterns: [
          {
            group: forbidden.flatMap((l) => [`@/${l}`, `@/${l}/*`]),
            message: `Нарушение границ FSD: "${layer}" не импортирует из вышестоящих слоёв. Пробрасывай зависимости вниз (DI).`,
          },
        ],
      },
    ],
  },
});

export default defineConfig([
  globalIgnores(['dist', '.next/**', 'node_modules/**']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      globals: globals.browser,
    },
    plugins: {
      'simple-import-sort': simpleImportSort,
    },
    rules: {
      // Вложенные тернарники читаются плохо — ранние return / хелпер-функция
      'no-nested-ternary': 'error',
      // FSD-сортировка импортов с пустой строкой между группами
      'simple-import-sort/imports': ['error', { groups: IMPORT_GROUPS }],
    },
  },
  {
    files: ['src/shared/ui/**/*.{ts,tsx}'],
    rules: {
      'react-refresh/only-export-components': 'off',
    },
  },
  ...Object.entries(LAYERS_ABOVE).map(([layer, forbidden]) =>
    fsdBoundary(layer, forbidden),
  ),
]);
