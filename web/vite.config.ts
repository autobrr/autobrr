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
      port: 3000
    },
    build: {
      // outDir: "build",
      manifest: true,
      // assetsDir: "static",
      // cssCodeSplit: true,
      sourcemap: true
      // rollupOptions: {
      //   output: {
      //     entryFileNames: "static/js/[name].[hash].js",
      //     chunkFileNames: "static/js/[name].[hash].js",
      //     assetFileNames: (assetInfo) => {
      //       const extension = assetInfo.name?.split(".").pop();
      //       switch (extension) {
      //       case "css":
      //         return "static/css/[name].[hash].css";
      //       case "woff2":
      //         return "static/media/[name].[hash].[ext]";
      //       default:
      //         return "static/[name].[hash].[ext]";
      //       }
      //     }
      //   }
      // }
      
    }
  });
};
