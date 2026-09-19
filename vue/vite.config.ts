import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { PrimeVueResolver } from '@primevue/auto-import-resolver'

// The host the dev stack is reachable on.
//
// One constant for both the allow-list and the HMR socket below, because they are the same
// fact and drifting apart breaks the page in a way that looks like a build problem: the
// compose service publishes no ports, so Traefik on this hostname is the *only* way in.
const DEV_HOST = 'hq.local.nathejk.dk'

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: '0.0.0.0',
    port: 80,
    // Where the *browser* should open the HMR websocket.
    //
    // Vite derives this from `server.host`/`server.port` by default, which here means it
    // tells the browser to connect to localhost:80 — the address inside the container, not
    // the one the page was loaded from. Through Traefik that is wrong twice over: wrong
    // host, and wrong scheme and port (the SPA is served over TLS on 443, and a browser in
    // a secure context refuses a plain ws:// socket anyway).
    //
    // Without this, HMR silently never connects, and that costs more than live reloading:
    // the socket is also how Vite tells a tab to reload itself after it re-optimizes
    // dependencies, so a tab that cannot hear it keeps requesting chunk URLs that no longer
    // exist and shows "The file does not exist at …/deps/chunk-….js" until somebody reloads
    // by hand. Diagnosed the hard way.
    hmr: {
      protocol: 'wss',
      host: DEV_HOST,
      clientPort: 443
    },
    proxy: {
      '/api': {
        target: 'http://api',
        changeOrigin: true
      }
    },
    allowedHosts: [DEV_HOST]
  },
  plugins: [
    vue(),
    Components({
      resolvers: [PrimeVueResolver()]
    })
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  }
})
