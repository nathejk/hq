import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'

// Standalone from vite.config.ts on purpose: unit tests here cover plain
// reactive logic, so they need neither the Vue SFC plugin nor a DOM. Add
// environment: 'jsdom' and the vue plugin only when component tests arrive.
export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/**/*.spec.ts'],
    // No watch by default: a watching process would hang a container run.
    watch: false,
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
