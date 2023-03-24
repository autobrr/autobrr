import { fileURLToPath, URL } from "node:url";

import react from "@vitejs/plugin-react-swc";
import { defineConfig, loadEnv } from "vite";
// https://vitejs.dev/config/
export default ({ mode }: { mode: any }) => {
  // early load .env file
  process.env = { ...process.env, ...loadEnv(mode, process.cwd()) };
  // import.meta.env.VITE_NAME available here with: process.env.VITE_NAME

  return defineConfig({
    plugins: [react()],
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url))
      }
    },
    server: {
      hmr: {
        overlay: true
      },
      port: 7474
    },
    build: {

      manifest: true,
      sourcemap: true      
    }
  });
};
