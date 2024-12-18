import { fileURLToPath, URL } from "node:url";
import { defineConfig, loadEnv, ConfigEnv } from "vite";
import { VitePWA } from "vite-plugin-pwa";
import react from "@vitejs/plugin-react-swc";
import svgr from "vite-plugin-svgr";
import path from "node:path";
import fs from "node:fs";

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
    // __BASE_URL__: "{{.BaseUrl}}",
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
    }),
    {
      name: "html-transformer-plugin",
      enforce: "post",
      apply: "build",
      async closeBundle() {
        const outputDir = 'dist'; // Adjust this if your `build.outDir` is different
        const htmlPath = path.resolve(outputDir, 'index.html');

        // Check if the file exists
        if (!fs.existsSync(htmlPath)) {
          console.error(`Could not find ${htmlPath}. Make sure the output directory matches.`);
          return;
        }

        // Read the `index.html` content
        let html = fs.readFileSync(htmlPath, 'utf-8');

        // Perform your transformations here
        html = html.replace('%7B%7B.BaseUrl%7D%7D/', '{{.BaseUrl}}'); // Example: Replace `{{.BaseUrl}}`

        // Write the updated `index.html` back
        fs.writeFileSync(htmlPath, html);

        console.log('Transformed index.html successfully.');
      },
    },
    ],
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
        },
        "/autobrr/api": {
          target: "http://127.0.0.1:7474/autobrr",
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
    },
    experimental: {
      renderBuiltUrl(filename: string, { hostId, hostType, type }: {
        hostId: string,
        hostType: 'js' | 'css' | 'html',
        type: 'public' | 'asset'
      }) {
        // console.debug(filename, hostId, hostType, type)
        return '{{.BaseUrl}}' + filename
        // if (type === 'public') {
        //   return 'https://www.domain.com/' + filename
        // }
        // else if (path.extname(hostId) === '.js') {
        //   return { runtime: `window.__assetsPath(${JSON.stringify(filename)})` }
        // }
        // else {
        //   return 'https://cdn.domain.com/assets/' + filename
        // }
      }
    }
  });
};
