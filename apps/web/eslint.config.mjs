import globals from 'globals';
import pluginJs from '@eslint/js';
import tseslint from 'typescript-eslint';
import pluginReact from 'eslint-plugin-react';
import pluginPrettier from 'eslint-plugin-prettier'; // Ajouter le plugin Prettier
import configPrettier from 'eslint-config-prettier'; // Importer la configuration Prettier


export default [
  {
    files: ['**/*.{js,mjs,cjs,ts,jsx,tsx}'],
    ignores: ['node_modules/', 'dist/'], // Ajoute les dossiers à ignorer ici
  },
  {
    files: ['**/*.js'],
    languageOptions: {
      sourceType: 'commonjs',
      globals: { ...globals.browser, ...globals.node }
    }
  },
  {
    files: ['**/*.ts', '**/*.tsx'],
    languageOptions: {
      parser: '@typescript-eslint/parser',
      parserOptions: {
        project: './tsconfig.json', // Spécifie le projet TypeScript
      },
      globals: { ...globals.browser, ...globals.node }
    },
  },
  pluginJs.configs.recommended,
  ...tseslint.configs.recommended,
  pluginReact.configs.flat.recommended,
  pluginPrettier.configs.recommended, // Ajouter les règles de Prettier
  configPrettier, // Ajouter la configuration Prettier pour désactiver les conflits
];
