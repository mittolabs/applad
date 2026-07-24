import js from '@eslint/js';
import globals from 'globals';
import tseslint from 'typescript-eslint';
import reactHooks from 'eslint-plugin-react-hooks';

/*
 * There was no lint config here at all, so `npm run lint` failed to even start
 * and the rules-of-hooks check never ran. That is how a `useMemo` placed after
 * an early return shipped: React counts hooks per render, a conditional one
 * changes the count, and it tears the whole tree down (error #310) rather than
 * failing the one component. The rule below catches that class at build time,
 * which is where it belongs.
 */
export default tseslint.config(
  { ignores: ['dist', 'node_modules', 'coverage'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2022,
      globals: { ...globals.browser, ...globals.es2022 },
    },
    plugins: { 'react-hooks': reactHooks },
    rules: {
      // The two that matter. Hook order is a correctness issue, not a style one.
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',

      // Keep the signal high: these are pervasive in this codebase and are not
      // what this config exists to catch.
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': [
        'warn',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      '@typescript-eslint/no-empty-object-type': 'off',
    },
  },
);
