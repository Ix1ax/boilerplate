import path from 'node:path';
import { FlatCompat } from '@eslint/eslintrc';
const compat = new FlatCompat({ baseDirectory: import.meta.dirname });
export default [
  {
    ignores: [
      '**/.next/**',
      '**/.next-dev/**',
      '**/node_modules/**',
      '**/next-env.d.ts',
    ],
  },
  ...compat.extends('next/core-web-vitals', 'next/typescript'),
  {
    settings: { next: { rootDir: path.join(import.meta.dirname, 'apps/web') } },
  },
];
