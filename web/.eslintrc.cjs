module.exports = {
  root: true,
  env: { browser: true, es2020: true },
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:react-hooks/recommended',
  ],
  ignorePatterns: ['dist', '.eslintrc.cjs'],
  parser: '@typescript-eslint/parser',
  plugins: [
    '@stylistic',
    'react-refresh'
  ],
  rules: {
    '@stylistic/comma-dangle': ["error", "never"],
    '@stylistic/indent': ['error', 2, { "SwitchCase": 1 }],
    '@stylistic/object-curly-spacing': ["error", "always"],
    '@typescript-eslint/no-unused-vars': ["warn"],
    'linebreak-style': ["error", "unix"],
    'react-refresh/only-export-components': ['warn', { allowConstantExport: true }]
  },
}
