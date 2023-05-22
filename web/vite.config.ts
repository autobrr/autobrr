import { fileURLToPath, URL } from "node:url";

import react from "@vitejs/plugin-react-swc";
import { VitePWA } from "vite-plugin-pwa";
import { defineConfig, loadEnv } from "vite";
// https://vitejs.dev/config/
export default ({ mode }: { mode: any }) => {
  // early load .env file
  process.env = { ...process.env, ...loadEnv(mode, process.cwd()) };
  // import.meta.env.VITE_NAME available here with: process.env.VITE_NAME

  return defineConfig({
    base: "",
    plugins: [react(), VitePWA({
      registerType: "autoUpdate",
      injectRegister: "inline",
      scope: "{{.BaseUrl}}",
      // strategies: "injectManifest",
      useCredentials: true,
      includeAssets: [
        "favicon.svg",
        "favicon.ico",
        "robots.txt",
        "logo.png",
        "apple-touch-icon-*.png",
        "manifest.webmanifest",
        "assets/**/*"
      ],
      manifest: {
        name: "autobrr",
        short_name: "autobrr",
        description: "Automation for downloads.",
        theme_color: "#141415",
        background_color: "#141415",
        icons: [
          {
            src: "logo.png",
            sizes: "192x192",
            type: "image/png"
          },
          {
            src: "logo192.png",
            sizes: "192x192",
            type: "image/png"
          },
          {
            src: "logo512.png",
            sizes: "512x512",
            type: "image/png"
          },
          {
            src: "logo512.png",
            sizes: "512x512",
            type: "image/png",
            purpose: "any maskable"
          },
          {
            src: "apple-touch-icon-iphone-retina-120x120.png",
            sizes: "120x120",
            type: "image/png"
          }
        ],
        start_url: "{{.BaseUrl}}",
        scope: "{{.BaseUrl}}",
        display: "standalone"
      },
      workbox: {
        globPatterns: ["**/*.{js,css,html,svg}"],
        sourcemap: true,
        navigateFallbackDenylist: [/^\/api/]
      }
    })],
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
    server: {
      port: 3000,
      hmr: {
        overlay: true
      },
    },
    build: {
      manifest: true,
      sourcemap: true
    }
  });
};
