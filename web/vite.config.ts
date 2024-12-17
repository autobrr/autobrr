import { fileURLToPath, URL } from "node:url";
import { defineConfig, loadEnv, ConfigEnv } from "vite";
import { VitePWA } from "vite-plugin-pwa";
import react from "@vitejs/plugin-react-swc";
import svgr from "vite-plugin-svgr";

interface PreRenderedAsset {
  name: string | undefined;
  source: string | Uint8Array;
  type: 'asset';
}

// https://vitejs.dev/config/
export default ({ mode }: ConfigEnv) => {
  // early load .env file
  // import.meta.env.VITE_NAME available here with: process.env.VITE_NAME
  process.env = { ...process.env, ...loadEnv(mode, process.cwd()) };

  return defineConfig({
    base: "",
    plugins: [react(), svgr(), VitePWA({
      injectRegister: null,
      selfDestroying: true,
      scope: "{{.BaseUrl}}",
      // strategies: "injectManifest",
      useCredentials: true,
      includeAssets: [
        // looks inside "public" folder 
        // manifest's icons are automatic added
        "favicon.ico"
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
            src: "apple-touch-icon-iphone-60x60.png",
            sizes: "60x60",
            type: "image/png"
          },
          {
            src: "apple-touch-icon-ipad-76x76.png",
            sizes: "76x76",
            type: "image/png"
          },
          {
            src: "apple-touch-icon-iphone-retina-120x120.png",
            sizes: "120x120",
            type: "image/png"
          },
          {
            src: "apple-touch-icon-ipad-retina-152x152.png",
            sizes: "152x152",
            type: "image/png"
          }
        ],
        start_url: "{{.BaseUrl}}",
        scope: "{{.BaseUrl}}",
        display: "standalone"
      },
      workbox: {
        // looks inside "dist" folder
        globPatterns: ["**/*.{js,css,html,svg,woff2}"],
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
      proxy: {
        "/api": {
          target: "http://127.0.0.1:7474/",
          changeOrigin: true,
          secure: false
        }
      }
    },
    build: {
      sourcemap: true,
      rollupOptions: {
        output: {
          assetFileNames: (chunkInfo: PreRenderedAsset) => {
            if (chunkInfo.name === "Inter-Variable.woff2") {
              return "assets/[name][extname]";
            }
            return "assets/[name]-[hash][extname]";
          }
        },
      }
    }
  });
};
