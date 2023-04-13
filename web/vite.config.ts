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
      injectRegister: "auto",
      includeAssets: [
        "favicon.svg",
        "favicon.ico",
        "robots.txt",
        "apple-touch-icon.png",
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
          }
        ],
        start_url: "/",
        display: "standalone"
        
      },
      workbox: {
        globPatterns: ["**/*.{js,css,html,svg}"],
        sourcemap: true
      }
    })],
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url))
      }
    },
    server: {
      port: 3000,
      hmr: {
        overlay: true
      },
      proxy: {
        "/api": {
          target: "http://127.0.0.1:7474/",
          changeOrigin: true,
          secure: false
        }
      }
    },
    build: {
      manifest: true,
      sourcemap: true
    }
  });
};