import tseslint from 'typescript-eslint';
export default [
  { ignores: ['**/node_modules/**', '**/dist/**', '**/.next/**'] },
  ...tseslint.configs.recommended,
];
