module.exports = {
  root: true,
  parser: "@typescript-eslint/parser",
  plugins: [
    "@typescript-eslint",
  ],
  // If we ever decide on a code-style, I'll leave this here.
  //extends: [
  //  "airbnb",
  //  "airbnb/hooks",
  //  "airbnb-typescript",
  //],
  rules: {
    // Turn off pesky "react not in scope" error while
    // we transition to proper ESLint support
    "react/react-in-jsx-scope": "off",
    // Add a UNIX-style linebreak at the end of each file
    "linebreak-style": ["error", "unix"],
    // Allow only double quotes and backticks
    quotes: ["error", "double"],
    // Warn if a line isn't indented with a multiple of 2
    indent: ["warn", 2, { "SwitchCase": 0 }],
    // Don't enforce any particular brace style
    curly: "off",
    // Allow only vars starting with _ to be ununsed vars
    "no-unused-vars": ["warn", {
      "varsIgnorePattern": "^_",
      "argsIgnorePattern": "^_",
      "caughtErrorsIgnorePattern": "^_",
      "ignoreRestSiblings": true
    }],
    // Let's keep these off for now and
    // maybe turn these back on sometime in the future
    "import/prefer-default-export": "off",
    "react/function-component-definition": "off",
    "nonblock-statement-body-position": ["warn", "below"]
  },
  // Conditionally run the following configuration only for TS files.
  // Otherwise, this will create inter-op problems with JS files.
  overrides: [
    {
      // Run only .ts and .tsx files
      files: ["*.ts", "*.tsx"],
      // Define the @typescript-eslint plugin schemas
      extends: [
        "plugin:@typescript-eslint/recommended",
        // Don't require strict type-checking for now, since we have too many
        // dubious statements literred in the code.
        //"plugin:@typescript-eslint/recommended-requiring-type-checking",
      ],
      parserOptions: {
        // project: "tsconfig.json",
        // This is needed so we can always point to the tsconfig.json
        // file relative to the current .eslintrc.js file.
        // Generally, a problem occurrs when "npm run lint"
        // gets ran from another directory. This fixes it.
        tsconfigRootDir: __dirname,
        sourceType: "module",
      },
      // Override JS rules and apply @typescript-eslint rules
      // as they might interfere with eachother.
      rules: {
        quotes: "off",
        "@typescript-eslint/quotes": ["error", "double"],
        semi: "off",
        "@typescript-eslint/semi": ["warn", "always"],
        indent: ["warn", 2, { "SwitchCase": 0 }],
        "@typescript-eslint/indent": "off",
        "@typescript-eslint/comma-dangle": "error",
        "keyword-spacing": "off",
        "@typescript-eslint/keyword-spacing": ["error"],
        "object-curly-spacing": "off",
        "@typescript-eslint/object-curly-spacing": ["warn", "always"],
        // Allow only vars starting with _ to be ununsed vars
        "@typescript-eslint/no-unused-vars": ["warn", {
          "varsIgnorePattern": "^_",
          "argsIgnorePattern": "^_",
          "caughtErrorsIgnorePattern": "^_",
          "ignoreRestSiblings": true
        }],
        // We have quite some "Unexpected any. Specify a different type" warnings.
        // This disables these warnings since they are false positives afaict.
        "@typescript-eslint/no-explicit-any": "off"
      },
    },
  ],
};
