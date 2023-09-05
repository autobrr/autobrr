module.exports = {
  root: true,
  parser: "@typescript-eslint/parser",
  plugins: ["@typescript-eslint", "prettier"],
  extends: [
    "plugin:@typescript-eslint/recommended",
    "prettier",
    "eslint-config-prettier" // Turn off ESLint formatting rules that would conflict with prettier
  ],
  rules: {
    // Turn off ESLint's semi rule; let prettier handle this
    semi: "off",
    // Focus more on code quality here
    "react/react-in-jsx-scope": "off",
    "no-unused-vars": [
      "warn",
      {
        varsIgnorePattern: "^_",
        argsIgnorePattern: "^_",
        caughtErrorsIgnorePattern: "^_",
        ignoreRestSiblings: true
      }
    ],
    "import/prefer-default-export": "off"
  },
  overrides: [
    {
      files: ["*.ts", "*.tsx"],
      extends: ["plugin:@typescript-eslint/recommended"],
      parserOptions: {
        tsconfigRootDir: __dirname,
        sourceType: "module"
      },
      rules: {
        // turn this off to avoid conflicts with prettier
        "prettier/prettier": "off",
        quotes: "off",
        "@typescript-eslint/quotes": ["error", "double"],
        semi: "off",
        // Let prettier handle this
        "@typescript-eslint/semi": "off",
        indent: "off",
        "@typescript-eslint/indent": "off",
        "@typescript-eslint/comma-dangle": ["error", "never"],
        "keyword-spacing": "off",
        "@typescript-eslint/keyword-spacing": "off",
        "object-curly-spacing": "off",
        "@typescript-eslint/object-curly-spacing": "off",
        "@typescript-eslint/no-unused-vars": [
          "warn",
          {
            varsIgnorePattern: "^_",
            argsIgnorePattern: "^_",
            caughtErrorsIgnorePattern: "^_",
            ignoreRestSiblings: true
          }
        ],
        "@typescript-eslint/no-explicit-any": "off"
      }
    }
  ]
};
