import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: {
    alias: [
      { find: "@", replacement: fileURLToPath(new URL("./src/", import.meta.url)) },
      { find: "@app", replacement: fileURLToPath(new URL("./src/", import.meta.url)) },
      { find: "@components", replacement: fileURLToPath(new URL("./src/components", import.meta.url)) },
      { find: "@forms", replacement: fileURLToPath(new URL("./src/forms", import.meta.url)) },
      { find: "@hooks", replacement: fileURLToPath(new URL("./src/hooks", import.meta.url)) },
      { find: "@api", replacement: fileURLToPath(new URL("./src/api", import.meta.url)) },
      { find: "@screens", replacement: fileURLToPath(new URL("./src/screens", import.meta.url)) },
      { find: "@utils", replacement: fileURLToPath(new URL("./src/utils", import.meta.url)) },
      { find: "@types", replacement: fileURLToPath(new URL("./src/types", import.meta.url)) },
      { find: "@domain", replacement: fileURLToPath(new URL("./src/domain", import.meta.url)) }
    ]
  },
  test: {
    environment: "jsdom",
    include: ["test/**/*.test.{ts,tsx}"],
    setupFiles: ["test/setup.ts"],
    env: {
      TZ: "UTC"
    }
  }
});
