module.exports = {
  root: true,
  parser: "@typescript-eslint/parser",
  plugins: ["@typescript-eslint", "prettier"],
  extends: [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
    "prettier",
  ],
  rules: {
    "prettier/prettier": "error",
    // Turn off ESLint rules that conflict with Prettier
    semi: "off",
    quotes: "off",
    indent: "off",
    "comma-dangle": "off",
    // Focus more on code quality here
    "react/react-in-jsx-scope": "off",
    "no-unused-vars": [
      "warn",
      {
        varsIgnorePattern: "^_",
        argsIgnorePattern: "^_",
        caughtErrorsIgnorePattern: "^_",
        ignoreRestSiblings: true,
      }
    ],
    "import/prefer-default-export": "off"
  },
  overrides: [
    {
      files: ["*.ts", "*.tsx"],
      extends: [
        "plugin:@typescript-eslint/recommended",
        // Uncomment this when you are ready for stricter type checking
        // "plugin:@typescript-eslint/recommended-requiring-type-checking",
      ],
      parserOptions: {
        tsconfigRootDir: __dirname,
        sourceType: "module",
      },
      rules: {
        "@typescript-eslint/quotes": ["error", "double"],
        "@typescript-eslint/semi": ["warn", "always"],
        "@typescript-eslint/indent": "off",
        "@typescript-eslint/comma-dangle": "warn",
        "@typescript-eslint/keyword-spacing": ["error"],
        "@typescript-eslint/object-curly-spacing": ["warn", "always"],
        "@typescript-eslint/no-unused-vars": ["warn", {
          "varsIgnorePattern": "^_",
          "argsIgnorePattern": "^_",
          "caughtErrorsIgnorePattern": "^_",
          "ignoreRestSiblings": true
        }],
        // Turn this on when we are ready for stricter type checking
        "@typescript-eslint/no-explicit-any": "off",
      },
    },
  ]
};