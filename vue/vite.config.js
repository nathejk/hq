import { fileURLToPath } from 'url'
import { defineConfig } from 'vite'
import { createVuePlugin as vue2 } from 'vite-plugin-vue2'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue2({
      jsx: true,
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      vue: 'vue/dist/vue.esm.js',
    },
  },
  build: {
    brotliSize: false, // unsupported in StackBlitz
  },
  server: {
    host: '0.0.0.0',
    port: 80,
    proxy: {
      '/api': {
        target: 'http://api',
        changeOrigin: true,
        //rewrite: (path) => path.replace(/^\/api/, '')
      },
      '^/ws$': {
        target: 'http://api',
        changeOrigin: true,
        ws: true,
      },
    },
  },
})
