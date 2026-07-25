const { defineConfig, globalIgnores } = require("eslint/config");
const js = require("@eslint/js");
const tseslint = require("@typescript-eslint/eslint-plugin");
const reactHooks = require("eslint-plugin-react-hooks");
const reactRefreshModule = require("eslint-plugin-react-refresh");
const globals = require("globals");

const reactRefresh = reactRefreshModule.default ?? reactRefreshModule;
const sourceFiles = ["src/**/*.{ts,tsx}"];
const tsFiles = ["**/*.{ts,tsx}"];
const nodeFiles = ["eslint.config.cjs", "vite.config.ts", "tailwind.config.ts"];

const withFiles = (config, files) => ({
  ...config,
  files,
});

module.exports = defineConfig([
  globalIgnores(["dist"]),
  withFiles(js.configs.recommended, ["eslint.config.cjs"]),
  ...tseslint.configs["flat/recommended"].map((config) => withFiles(config, tsFiles)),
  {
    files: sourceFiles,
    languageOptions: {
      globals: {
        ...globals.browser,
      },
    },
  },
  {
    files: nodeFiles,
    languageOptions: {
      globals: {
        ...globals.node,
      },
    },
  },
  {
    files: sourceFiles,
    plugins: {
      "react-hooks": reactHooks,
    },
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
    },
  },
  {
    ...reactRefresh.configs.vite,
    files: sourceFiles,
    rules: {
      ...reactRefresh.configs.vite.rules,
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
    },
  },
]);
